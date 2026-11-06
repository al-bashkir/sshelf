package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// customHost opens the connect prompt for group index gi (-1: no group),
// opened from Groups, in a 60x12 modal.
func customHost(gi int) *customHostModel {
	m := newCustomHostModel(gi, screenGroups)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	return m
}

// typeText types s into m one key at a time.
func typeText(m tea.Model, s string) {
	for _, r := range s {
		sendKeys(m, string(r))
	}
}

func TestCustomHostKeys(t *testing.T) {
	cases := []struct {
		name  string
		gi    int
		typed string
		key   string
		want  tea.Msg // nil: no message and a "host required" warning
	}{
		{"esc cancels", 0, "a", "esc", customHostCancelMsg{}},
		{"enter connects to the typed hosts", 1, " a  b ", "enter",
			customHostConnectMsg{returnTo: screenGroups, groupIndex: 1, hosts: []string{"a", "b"}}},
		{"enter without a group", -1, "a", "enter",
			customHostConnectMsg{returnTo: screenGroups, groupIndex: -1, hosts: []string{"a"}}},
		{"enter needs a host", 0, "  ", "enter", nil},
		{"ctrl+g adds to the group", 1, "a b", "ctrl+g",
			customHostDoneMsg{returnTo: screenGroups, groupIndex: 1, hosts: []string{"a", "b"}}},
		{"ctrl+g adds to group 0", 0, "a", "ctrl+g",
			customHostDoneMsg{returnTo: screenGroups, groupIndex: 0, hosts: []string{"a"}}},
		{"ctrl+g without a group picks one", -1, "a", "ctrl+g",
			customHostPickGroupMsg{returnTo: screenGroups, hosts: []string{"a"}}},
		{"ctrl+g needs a host", -1, " ", "ctrl+g", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := customHost(c.gi)
			typeText(m, c.typed)
			_, cmd := m.Update(hk(c.key))
			if c.want == nil {
				if cmd != nil {
					t.Fatalf("got %#v, want no message", cmd())
				}
				screenShows(t, m, []string{"host required"})
				return
			}
			if cmd == nil {
				t.Fatalf("no message, want %#v", c.want)
			}
			if got := cmd(); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v, want %#v", got, c.want)
			}
			screenShows(t, m, nil, "host required")
		})
	}
}

func TestCustomHostClearsTheWarningOnSubmit(t *testing.T) {
	for _, k := range []string{"enter", "ctrl+g"} {
		t.Run(k, func(t *testing.T) {
			m := customHost(0)
			sendKeys(m, k)
			typeText(m, "a")
			screenShows(t, m, []string{"host required"}) // typing keeps it
			sendKeys(m, k)
			screenShows(t, m, nil, "host required")
		})
	}
}

// Ctrl+a belongs to the text field: it moves the cursor to the line start.
func TestCustomHostCtrlAMovesToLineStart(t *testing.T) {
	m := customHost(0)
	typeText(m, "ab")
	m.Update(hk("ctrl+a"))
	typeText(m, "x")
	if got := m.input.Value(); got != "xab" {
		t.Fatalf("input %q, want %q", got, "xab")
	}
}

func TestCustomHostView(t *testing.T) {
	m := customHost(0)
	m.parentCrumb = "Groups > web"
	screenShows(t, m, []string{"Groups > web > Connect", "host1 user@host2 ...",
		"⏎ connect  Ctrl+g add to group  Esc cancel"})
	typeText(m, "ab")
	screenShows(t, m, []string{"host: ab"}, "host1 user@host2")
}

func TestCustomHostInputFitsTheModal(t *testing.T) {
	for _, c := range []struct{ w, want int }{
		{60, 50}, // 60 - borders and padding (4) - "host: " (6)
		{20, 10},
		{16, 10}, // never under 10
	} {
		m := customHost(0)
		m.Update(tea.WindowSizeMsg{Width: c.w, Height: 12})
		if m.input.Width != c.want {
			t.Fatalf("width %d: input %d wide, want %d", c.w, m.input.Width, c.want)
		}
	}
}

func TestCustomHostTakes512Characters(t *testing.T) {
	m := customHost(0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(strings.Repeat("a", 600))})
	if n := len(m.input.Value()); n != 512 {
		t.Fatalf("input holds %d characters, want 512", n)
	}
}
