package tmux

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

// fakeTmux records tmux argv and answers like a tmux server: new-window
// prints "@1 %1", each split-window prints the next pane id.
type fakeTmux struct {
	calls    [][]string
	panes    int
	newOut   string // overrides the new-window output when set
	splitOut *string
	failOn   string // subcommand that fails with output "boom"
}

func (f *fakeTmux) run(args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if args[0] == f.failOn {
		return []byte("boom\n"), errors.New("exit status 1")
	}
	switch args[0] {
	case "new-window":
		if f.newOut != "" {
			return []byte(f.newOut), nil
		}
		return []byte("@1 %1\n"), nil
	case "split-window":
		if f.splitOut != nil {
			return []byte(*f.splitOut), nil
		}
		f.panes++
		return []byte(fmt.Sprintf("%%%d\n", f.panes+1)), nil
	}
	return nil, nil
}

var (
	ssh1 = []string{"ssh", "a"}
	ssh2 = []string{"ssh", "b"}
	ssh3 = []string{"ssh", "c"}
)

func swo(kv ...string) []string {
	return append([]string{"set-window-option", "-t", "@1"}, kv...)
}

func TestOpenOneWindowDefaults(t *testing.T) {
	f := &fakeTmux{}
	if err := openOneWindow(f.run, [][]string{ssh1, ssh2}, OneWindowOpts{SplitFlag: " x "}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"new-window", "-P", "-F", "#{window_id} #{pane_id}", "-n", "ssh", "--", "ssh", "a"},
		swo("automatic-rename", "off"),
		swo("allow-rename", "off"),
		swo("pane-border-status", "bottom"),
		swo("pane-border-format", config.DefaultPaneBorderFormat),
		{"split-window", "-t", "@1", "-h", "-P", "-F", "#{pane_id}", "--", "ssh", "b"},
		{"select-layout", "-t", "@1", "even-horizontal"},
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls:\n%q\nwant:\n%q", f.calls, want)
	}
}

func TestOpenOneWindowAllOptions(t *testing.T) {
	f := &fakeTmux{}
	opts := OneWindowOpts{
		WindowName:       " web ",
		PaneTitles:       []string{" a ", "b"}, // no title for the third pane
		SplitFlag:        "-v",
		Layout:           "tiled",
		SyncPanes:        true,
		PaneBorderStatus: "OFF",
		PaneBorderFormat: "ignored when off",
	}
	if err := openOneWindow(f.run, [][]string{ssh1, ssh2, ssh3}, opts); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"new-window", "-P", "-F", "#{window_id} #{pane_id}", "-n", "web", "--", "ssh", "a"},
		swo("automatic-rename", "off"),
		swo("allow-rename", "off"),
		swo("pane-border-status", "off"),
		swo("synchronize-panes", "on"),
		{"select-pane", "-t", "%1", "-T", "a"},
		{"split-window", "-t", "@1", "-v", "-P", "-F", "#{pane_id}", "--", "ssh", "b"},
		{"select-pane", "-t", "%2", "-T", "b"},
		{"split-window", "-t", "@1", "-v", "-P", "-F", "#{pane_id}", "--", "ssh", "c"},
		{"select-layout", "-t", "@1", "tiled"},
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls:\n%q\nwant:\n%q", f.calls, want)
	}
}

func TestOpenOneWindowBorderFormatAndSingleHostSync(t *testing.T) {
	f := &fakeTmux{}
	opts := OneWindowOpts{SyncPanes: true, PaneBorderStatus: "top", PaneBorderFormat: "#{pane_title}"}
	if err := openOneWindow(f.run, [][]string{ssh1}, opts); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if reflect.DeepEqual(c, swo("synchronize-panes", "on")) {
			t.Fatal("synchronize-panes set for a single pane")
		}
	}
	if !reflect.DeepEqual(f.calls[3:5], [][]string{swo("pane-border-status", "top"), swo("pane-border-format", "#{pane_title}")}) {
		t.Fatalf("border calls: %q", f.calls[3:5])
	}
}

func TestOpenOneWindowSplitWithoutPaneIDSkipsTitle(t *testing.T) {
	empty := " \n"
	f := &fakeTmux{splitOut: &empty}
	opts := OneWindowOpts{PaneTitles: []string{"", "b"}}
	if err := openOneWindow(f.run, [][]string{ssh1, ssh2}, opts); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if c[0] == "select-pane" {
			t.Fatalf("unexpected title call %q", c)
		}
	}
}

func TestOpenOneWindowErrors(t *testing.T) {
	tc := []struct {
		name string
		cmds [][]string
		f    *fakeTmux
		want string
	}{
		{"no hosts", nil, &fakeTmux{}, "no hosts selected"},
		{"new-window fails", [][]string{ssh1}, &fakeTmux{failOn: "new-window"}, "tmux error: boom"},
		{"new-window ids missing", [][]string{ssh1}, &fakeTmux{newOut: "@1\n"}, "tmux error: missing window/pane id"},
		{"split fails", [][]string{ssh1, ssh2}, &fakeTmux{failOn: "split-window"}, "tmux error: boom"},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			err := openOneWindow(c.f.run, c.cmds, OneWindowOpts{})
			if err == nil || err.Error() != c.want {
				t.Fatalf("err=%v, want %q", err, c.want)
			}
			if c.name == "no hosts" && len(c.f.calls) != 0 {
				t.Fatalf("tmux called: %q", c.f.calls)
			}
			if c.name == "split fails" && c.f.calls[len(c.f.calls)-1][0] != "split-window" {
				t.Fatalf("ran tmux after the failed split: %q", c.f.calls)
			}
		})
	}
}

func TestTmuxErrMsgFallsBackToError(t *testing.T) {
	if got := tmuxErrMsg([]byte(" \n"), errors.New("exit status 1")); got != "exit status 1" {
		t.Fatalf("got %q", got)
	}
	if got := tmuxErrMsg([]byte(" boom \n"), errors.New("x")); !strings.EqualFold(got, "boom") {
		t.Fatalf("got %q", got)
	}
}
