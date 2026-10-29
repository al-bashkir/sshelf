package connect

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
	tmx "github.com/al-bashkir/sshelf/internal/tmux"
)

func TestModeGroupOverridesDefaults(t *testing.T) {
	d := config.Defaults{Tmux: "auto", OpenMode: "current"}
	tests := []struct {
		name  string
		group *config.Group
		want  tmx.OpenMode
	}{
		{"nil group", nil, tmx.OpenCurrent},
		{"blank group keeps defaults", &config.Group{Tmux: " ", OpenMode: "  "}, tmx.OpenCurrent},
		{"group force + pane", &config.Group{Tmux: "force", OpenMode: "tmux-pane"}, tmx.OpenPane},
		{"group open_mode only", &config.Group{OpenMode: "tmux-window"}, tmx.OpenWindow},
		{"group tmux only", &config.Group{Tmux: "force"}, tmx.OpenWindow},
	}
	for _, tc := range tests {
		if got, inTmux := Mode(d, tc.group, false); got != tc.want || inTmux {
			t.Errorf("%s: got %s,%v want %s,false", tc.name, got, inTmux, tc.want)
		}
	}
}

func TestModeReportsInTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/x,1,0")
	if got, in := Mode(config.Defaults{Tmux: "auto", OpenMode: "auto"}, nil, false); got != tmx.OpenWindow || !in {
		t.Fatalf("got %s,%v want tmux-window,true", got, in)
	}
}

func TestNeedsTmux(t *testing.T) {
	tests := []struct {
		mode   tmx.OpenMode
		inTmux bool
		want   bool
	}{
		{tmx.OpenCurrent, false, false},
		{tmx.OpenCurrent, true, false},
		{tmx.OpenWindow, false, false},
		{tmx.OpenWindow, true, true},
		{tmx.OpenPane, false, false},
		{tmx.OpenPane, true, true},
	}
	for _, tc := range tests {
		if got := NeedsTmux(tc.mode, tc.inTmux); got != tc.want {
			t.Errorf("%s inTmux=%v: got %v want %v", tc.mode, tc.inTmux, got, tc.want)
		}
	}
}

// fakeTmux puts a tmux stand-in first in PATH. It records each call's argv
// (one arg per line) in the returned file and exits with code.
func fakeTmux(t *testing.T, code string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> '" + log + "'\nprintf -- '--call--\\n' >> '" + log + "'\nexit " + code + "\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o700); err != nil { // #nosec G306 -- test stub must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestOpenSingleWindowNamesWindowAfterHost(t *testing.T) {
	log := fakeTmux(t, "0")
	_, msg, err := Open([]string{"web01"}, [][]string{{"ssh", "web01"}}, config.Defaults{}, nil, tmx.OpenWindow, true)
	if err != nil || msg != "opened 1" {
		t.Fatalf("msg=%q err=%v", msg, err)
	}
	data, _ := os.ReadFile(log)
	args := strings.Split(strings.TrimSpace(string(data)), "\n")
	if !reflect.DeepEqual(args[:4], []string{"new-window", "-n", "web01", "--"}) {
		t.Fatalf("tmux args=%q", args)
	}
	if !slicesContain(args, "ssh") {
		t.Fatalf("ssh argv missing: %q", args)
	}
}

func TestOpenSingleWindowGroupNameWins(t *testing.T) {
	log := fakeTmux(t, "0")
	g := &config.Group{Name: "prod"}
	if _, _, err := Open([]string{"web01"}, [][]string{{"ssh", "web01"}}, config.Defaults{}, g, tmx.OpenWindow, true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	if !strings.HasPrefix(string(data), "new-window\n-n\nprod\n") {
		t.Fatalf("tmux args=%q", data)
	}
}

func TestOpenSingleWindowTmuxFailure(t *testing.T) {
	fakeTmux(t, "1")
	_, _, err := Open([]string{"web01"}, [][]string{{"ssh", "web01"}}, config.Defaults{}, nil, tmx.OpenWindow, true)
	if err == nil || !strings.Contains(err.Error(), "tmux new-window") {
		t.Fatalf("err=%v", err)
	}
}

func slicesContain(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
