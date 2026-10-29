package sshcmd

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestApplyGroupOverDefaults(t *testing.T) {
	d := config.Defaults{User: "", Port: 22, IdentityFile: "", ExtraArgs: []string{"-o", "A=B"}}
	g := config.Group{User: "root", Port: 0, IdentityFile: "id", ExtraArgs: nil, RemoteCommand: ""}

	got := ApplyGroup(FromDefaults(d), g)
	want := Settings{User: "root", Port: 22, IdentityFile: "id", ExtraArgs: []string{"-o", "A=B"}, RemoteCommand: ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v, want %#v", got, want)
	}
}

func TestBuildCommandRemoteCommand(t *testing.T) {
	cmd := BuildCommand("example.com", Settings{RemoteCommand: "echo hi"})
	want := []string{"ssh", "--", "example.com", "sh -c 'echo hi'"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandPlainHost(t *testing.T) {
	cmd := BuildCommand("example.com", Settings{User: "me", Port: 2222, IdentityFile: "~/.ssh/id", ExtraArgs: []string{"-o", "StrictHostKeyChecking=no"}})
	want := []string{"ssh", "-i", "~/.ssh/id", "-p", "2222", "-o", "StrictHostKeyChecking=no", "--", "me@example.com"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandBracketHostForcesPort(t *testing.T) {
	cmd := BuildCommand("[10.10.10.10]:2201", Settings{User: "", Port: 2222})
	want := []string{"ssh", "-p", "2201", "--", "10.10.10.10"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandPort22Omitted(t *testing.T) {
	cmd := BuildCommand("example.com", Settings{Port: 22})
	want := []string{"ssh", "--", "example.com"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandHostIsNeverAnOption(t *testing.T) {
	cmd := BuildCommand("-oProxyCommand=touch /tmp/x", Settings{})
	want := []string{"ssh", "--", "-oProxyCommand=touch /tmp/x"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestApplyGroupIgnoresBlankFieldsLikeApplyHost(t *testing.T) {
	base := Settings{User: "me", IdentityFile: "id"}
	g := config.Group{User: "  ", IdentityFile: " "}
	if got := ApplyGroup(base, g); !reflect.DeepEqual(got, base) {
		t.Fatalf("blank group fields overrode defaults: got=%#v want %#v", got, base)
	}
}

func TestFindHostConfig(t *testing.T) {
	hosts := []config.Host{{Host: "web", User: "w"}, {Host: " db ", User: "d"}}
	for _, tt := range []struct {
		host, wantUser string
		ok             bool
	}{
		{"web", "w", true},
		{"db", "d", true},          // entry is trimmed
		{"[web]:2222", "w", true},  // known_hosts bracket form falls back to the bare host
		{"[mail]:2222", "", false}, // no entry for the bare host either
		{"  ", "", false},          // blank host never matches
	} {
		got, ok := FindHostConfig(hosts, tt.host)
		if ok != tt.ok || got.User != tt.wantUser {
			t.Errorf("FindHostConfig(%q) = (%q, %v), want (%q, %v)", tt.host, got.User, ok, tt.wantUser, tt.ok)
		}
	}
}
