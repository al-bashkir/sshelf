package connect

import (
	"os"
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/sshcmd"
	tmx "github.com/al-bashkir/sshelf/internal/tmux"
)

func TestOpenCurrentExecsSSHDirectly(t *testing.T) {
	ssh := []string{"ssh", "web01"}
	got, _, err := Open([]string{"web01"}, [][]string{ssh}, config.Defaults{}, nil, tmx.OpenCurrent, false)
	if err != nil || !reflect.DeepEqual(got, ssh) {
		t.Fatalf("got=%v err=%v want=%v", got, err, ssh)
	}
}

func TestOpenTmuxWrapsSSHForReconnect(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	got, _, err := Open([]string{"web01"}, [][]string{{"ssh", "web01"}}, config.Defaults{TmuxSession: "s"}, nil, tmx.OpenWindow, false)
	want := []string{"tmux", "new-session", "-s", "s", "--", exe, "__pane", "web01", "--", "ssh", "web01"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v err=%v want=%v", got, err, want)
	}
}

func TestOpenMultiHostNeedsTmux(t *testing.T) {
	cmds := [][]string{{"ssh", "a"}, {"ssh", "b"}}
	for _, mode := range []tmx.OpenMode{tmx.OpenCurrent, tmx.OpenWindow, tmx.OpenPane} {
		if _, _, err := Open([]string{"a", "b"}, cmds, config.Defaults{}, nil, mode, false); err == nil {
			t.Errorf("mode=%s outside tmux: want error", mode)
		}
	}
}

func TestUsesOneWindow(t *testing.T) {
	tc := []struct {
		mode  tmx.OpenMode
		hosts int
		want  bool
	}{
		{tmx.OpenPane, 1, true},
		{tmx.OpenPane, 3, true},
		{tmx.OpenWindow, 1, false},
		{tmx.OpenWindow, 2, true},
	}
	for _, tt := range tc {
		if got := usesOneWindow(tt.mode, tt.hosts); got != tt.want {
			t.Errorf("usesOneWindow(%s, %d)=%v want=%v", tt.mode, tt.hosts, got, tt.want)
		}
	}
}

func TestTmuxIsUnreachableFromTests(t *testing.T) {
	_, _, err := Open([]string{"a", "b"}, [][]string{{"ssh", "a"}, {"ssh", "b"}}, config.Defaults{}, nil, tmx.OpenPane, true)
	if err == nil {
		t.Fatal("Open reached a tmux server from tests; panes may have been created")
	}
}

func TestModeNoTmuxBeatsGroupForce(t *testing.T) {
	d := config.Defaults{Tmux: "force", OpenMode: "tmux-window"}
	if got, _ := Mode(d, &config.Group{Tmux: "force"}, true); got != tmx.OpenCurrent {
		t.Fatalf("noTmux: got %s want %s", got, tmx.OpenCurrent)
	}
	if got, _ := Mode(d, &config.Group{Tmux: "force"}, false); got != tmx.OpenWindow {
		t.Fatalf("tmux allowed: got %s want %s", got, tmx.OpenWindow)
	}
}

// The merge order is documented in docs/functional/config.md and README.md.
func TestBuildCommandsMergeOrder(t *testing.T) {
	d := config.Defaults{User: "def", Port: 22}
	inv := config.Inventory{Hosts: []config.Host{{Host: "web", User: "hostuser", Port: 2200}}}
	group := &config.Group{User: "groupuser"}

	got := BuildCommands([]string{"web", "db"}, d, inv, group, func(s *sshcmd.Settings) { s.IdentityFile = "k" })
	want := [][]string{
		{"ssh", "-i", "k", "-p", "2200", "--", "groupuser@web"}, // host port kept, group user wins
		{"ssh", "-i", "k", "--", "groupuser@db"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	if got := BuildCommands([]string{"web"}, d, inv, nil, nil); !reflect.DeepEqual(got, [][]string{{"ssh", "-p", "2200", "--", "hostuser@web"}}) {
		t.Fatalf("without group: got %v", got)
	}
}
