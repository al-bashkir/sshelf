package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/al-bashkir/sshelf/internal/config"
)

// A modal frame takes exactly the size it is given, border included.
func TestFocusedFrameFitsItsSize(t *testing.T) {
	lines := strings.Split(renderFocusedFrame(40, 10, "Title", "", "body", "footer"), "\n")
	if len(lines) != 10 {
		t.Fatalf("frame is %d lines, want 10", len(lines))
	}
	for _, l := range lines {
		if w := lipgloss.Width(l); w != 40 {
			t.Fatalf("line %d wide, want 40: %q", w, l)
		}
	}
}

// With a long query, the search box fills exactly the room joinHeader leaves
// next to the right side: one column more and the cursor cell is cut off.
func TestSearchHeaderFillsTheRoom(t *testing.T) {
	in := newSearchInput()
	in.Focus()
	in.SetValue(strings.Repeat("z", 200))
	for _, w := range []int{50, 80, 120} {
		for _, right := range []string{"", "● 3 hosts", renderToast(toast{text: "a long toast here", level: toastOK})} {
			room := max(0, w-2) - lipgloss.Width(right) - 1
			if got := lipgloss.Width(searchHeader(&in, w, right)); got != room {
				t.Errorf("width %d, right %q: search view %d wide, want %d", w, right, got, room)
			}
		}
	}
}

// A long query keeps its end (where the cursor is) in view, however wide
// the counts or toast on the right of the header are.
func TestSearchKeepsCursorInView(t *testing.T) {
	query := strings.Repeat("z", 120) + "END"
	long := toast{text: "a long toast that takes header room", level: toastOK}

	hosts := hostsScreen(nil)
	on(hosts, "space", "/", query) // "1 selected" widens the right side

	groups := groupsModelWith("alpha")
	groups.toast = long
	on(groups, "/", query)

	inv := config.Inventory{Groups: []config.Group{{Name: "prod", Hosts: []string{"a"}}}}
	gh := newGroupHostsModel(Options{Config: config.DefaultConfig(), Inventory: inv, Hosts: []string{"a"}}, 0)
	gh.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	gh.toast = long
	on(gh, "/", query)

	for name, v := range map[string]string{"hosts": hosts.View(), "groups": groups.View(), "group hosts": gh.View()} {
		if !strings.Contains(v, "END") {
			t.Errorf("%s: query end cut off:\n%s", name, v)
		}
	}
	if !strings.Contains(hosts.View(), "1 selected") {
		t.Error("hosts: selection count cut off")
	}
}
