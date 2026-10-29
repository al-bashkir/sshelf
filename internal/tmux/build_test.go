package tmux

import (
	"reflect"
	"testing"
)

func TestResolveOpenMode(t *testing.T) {
	tc := []struct {
		name     string
		tmux     string
		openMode string
		inTmux   bool
		want     OpenMode
	}{
		{"never forces current", "never", "tmux-window", true, OpenCurrent},
		{"force defaults to window", "force", "current", false, OpenWindow},
		{"force pane", "force", "tmux-pane", false, OpenPane},
		{"auto not in tmux defaults current", "auto", "auto", false, OpenCurrent},
		{"auto in tmux defaults window", "auto", "auto", true, OpenWindow},
		{"explicit window", "auto", "tmux-window", false, OpenWindow},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOpenMode(tt.tmux, tt.openMode, tt.inTmux)
			if got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestCmdBuilders(t *testing.T) {
	ssh := []string{"ssh", "-p", "2222", "host"}

	if got, want := NewWindowCmd("h", ssh), []string{"tmux", "new-window", "-n", "h", "--", "ssh", "-p", "2222", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NewWindowCmd=%v want=%v", got, want)
	}
}

func TestNewSessionCmd(t *testing.T) {
	ssh := []string{"ssh", "host"}
	exists := func(want bool) func(string) bool { return func(string) bool { return want } }

	if got, want := NewSessionCmd("sess", ssh, exists(false)), []string{"tmux", "new-session", "-s", "sess", "--", "ssh", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("new session: got %v want %v", got, want)
	}
	// An existing session must get a new window; "new-session -A" would attach and drop the command.
	if got, want := NewSessionCmd("sess", ssh, exists(true)), []string{"tmux", "new-window", "-t", "=sess:", "--", "ssh", "host", ";", "attach-session", "-t", "=sess"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("existing session: got %v want %v", got, want)
	}

	var asked string
	NewSessionCmd("  ", ssh, func(s string) bool { asked = s; return false })
	if asked != "sshelf" {
		t.Fatalf("blank session name checked as %q, want sshelf", asked)
	}
}
