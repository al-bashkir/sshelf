package tmux

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestInTmux(t *testing.T) {
	t.Setenv("TMUX", "")
	if InTmux() {
		t.Error("empty TMUX: want false")
	}
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")
	if !InTmux() {
		t.Error("TMUX set: want true")
	}
}

// Never reach the user's tmux server from tests.
func TestHasSessionWithoutServer(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_TMPDIR", t.TempDir())
	if HasSession("sshelf") {
		t.Fatal("no server: want false")
	}
}

func TestResolvePaneSettingsDefaults(t *testing.T) {
	d := config.Defaults{PaneSplit: " vertical ", PaneLayout: "tiled", PaneSync: "on", PaneBorderFmt: " fmt ", PaneBorderPos: " top "}
	got := ResolvePaneSettings(d, nil, 2)
	want := PaneSettings{SplitFlag: "-v", Layout: "tiled", SyncPanes: true, BorderFormat: "fmt", BorderStatus: "top"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want %#v", got, want)
	}
}

func TestResolvePaneSettingsGroupOverrides(t *testing.T) {
	d := config.Defaults{PaneSplit: "vertical", PaneLayout: "tiled", PaneSync: "on", PaneBorderFmt: "d", PaneBorderPos: "top"}
	g := &config.Group{PaneSplit: "horizontal", PaneLayout: "mv", PaneSync: "off", PaneBorderFmt: "g", PaneBorderPos: "bottom"}
	got := ResolvePaneSettings(d, g, 2)
	want := PaneSettings{SplitFlag: "-h", Layout: "main-vertical", SyncPanes: false, BorderFormat: "g", BorderStatus: "bottom"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want %#v", got, want)
	}
	// Blank group fields keep the defaults.
	got = ResolvePaneSettings(d, &config.Group{PaneSplit: " ", PaneLayout: " ", PaneSync: " ", PaneBorderFmt: " ", PaneBorderPos: " "}, 2)
	want = PaneSettings{SplitFlag: "-v", Layout: "tiled", SyncPanes: true, BorderFormat: "d", BorderStatus: "top"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("blank group: got=%#v want %#v", got, want)
	}
}

func TestResolvePaneSettingsSplitFlag(t *testing.T) {
	for in, want := range map[string]string{"vertical": "-v", "V": "-v", " v ": "-v", "horizontal": "-h", "": "-h", "x": "-h"} {
		if got := ResolvePaneSettings(config.Defaults{PaneSplit: in}, nil, 2).SplitFlag; got != want {
			t.Errorf("split %q: got %s want %s", in, got, want)
		}
	}
}

func TestResolvePaneSettingsLayout(t *testing.T) {
	tests := []struct {
		layout string
		panes  int
		want   string
	}{
		{"", 3, "even-vertical"}, {"auto", 3, "even-vertical"},
		{"", 4, "tiled"}, {"AUTO", 5, "tiled"},
		{"t", 2, "tiled"}, {"eh", 2, "even-horizontal"}, {"even-horizontal", 2, "even-horizontal"},
		{"ev", 2, "even-vertical"}, {"mh", 2, "main-horizontal"}, {"mv", 2, "main-vertical"},
		{" custom-layout ", 2, "custom-layout"},
	}
	for _, tc := range tests {
		if got := ResolvePaneSettings(config.Defaults{PaneLayout: tc.layout}, nil, tc.panes).Layout; got != tc.want {
			t.Errorf("layout %q panes=%d: got %q want %q", tc.layout, tc.panes, got, tc.want)
		}
	}
}

func TestResolvePaneSettingsSync(t *testing.T) {
	for in, want := range map[string]bool{"": true, "on": true, "yes": true, "off": false, "OFF": false, "false": false, "0": false, "no": false, " no ": false} {
		if got := ResolvePaneSettings(config.Defaults{PaneSync: in}, nil, 2).SyncPanes; got != want {
			t.Errorf("sync %q: got %v want %v", in, got, want)
		}
	}
}
