package ui

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestExitResult(t *testing.T) {
	ssh := []string{"ssh", "web01"}
	boom := errors.New("boom")

	// A program error wins: never exec after the TUI failed.
	if err := exitResult(&appModel{execCmd: ssh}, boom); !errors.Is(err, boom) {
		t.Fatalf("program error: got %v want %v", err, boom)
	}
	var req *ExecRequest
	if err := exitResult(&appModel{execCmd: ssh}, nil); !errors.As(err, &req) || !reflect.DeepEqual(req.Cmd, ssh) {
		t.Fatalf("exec: got %v want ExecRequest%v", err, ssh)
	}
	if err := exitResult(&appModel{quitting: true}, nil); !errors.Is(err, ErrQuit) {
		t.Fatalf("quit: got %v want ErrQuit", err)
	}
	if err := exitResult(&appModel{}, nil); err != nil {
		t.Fatalf("plain end: got %v want nil", err)
	}
}

func TestWithDefaultKnownHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	load := config.Config{Defaults: config.Defaults{LoadKnownHosts: true}}

	tc := []struct {
		name string
		in   Options
		want []string
	}{
		{"load, no path: default", Options{Config: load}, []string{filepath.Join(home, ".ssh", "known_hosts")}},
		{"load, path given: kept", Options{Config: load, KnownHosts: []string{"/etc/ssh/ssh_known_hosts"}}, []string{"/etc/ssh/ssh_known_hosts"}},
		{"no load: none", Options{}, nil},
	}
	for _, tt := range tc {
		if got := withDefaultKnownHosts(tt.in).KnownHosts; !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: KnownHosts=%q want %q", tt.name, got, tt.want)
		}
	}
}
