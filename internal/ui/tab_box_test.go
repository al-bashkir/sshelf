package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/al-bashkir/sshelf/internal/config"
)

// tabbFrame checks the frame of a w x h tab box view: the top and bottom
// borders, the tabs row, a separator on each line in seps, blank rows from
// blankFrom to blankTo, and a border on both sides of every other row.
func tabbFrame(t *testing.T, name, view string, w, h int, seps []int, blankFrom, blankTo int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != h {
		t.Fatalf("%s: %d lines, want %d:\n%s", name, len(lines), h, view)
	}
	dashes, spaces := strings.Repeat("─", w-2), strings.Repeat(" ", w-2)
	want := map[int]string{
		0:     "┌" + dashes + "┐",
		1:     "│Hosts  Groups  Settings" + spaces[len("Hosts  Groups  Settings"):] + "│",
		h - 1: "└" + dashes + "┘",
	}
	for _, i := range seps {
		want[i] = "├" + dashes + "┤"
	}
	for i := blankFrom; i <= blankTo; i++ {
		want[i] = "│" + spaces + "│"
	}
	for i, l := range lines {
		s, ok := want[i]
		switch {
		case ok && l != s:
			t.Errorf("%s: line %d:\n got %q\nwant %q", name, i, l, s)
		case !ok && (!strings.HasPrefix(l, "│") || !strings.HasSuffix(l, "│")):
			t.Errorf("%s: line %d has no side borders: %q", name, i, l)
		}
		if lipgloss.Width(l) != w {
			t.Errorf("%s: line %d is %d cells, want %d: %q", name, i, lipgloss.Width(l), w, l)
		}
	}
}

// The list screens draw the tab box over the whole window: the tabs row,
// the header row, the rows, blank rows to the footer, the footer.
func TestTabbScreens(t *testing.T) {
	tabbFrame(t, "hosts", plainView(hostsScreen(nil)), 100, 30, []int{2, 4, 26}, 8, 25)
	tabbFrame(t, "groups", plainView(groupsScreen(nil)), 100, 30, []int{2, 4, 26}, 8, 25)

	// Settings has no box footer: its key hints are its last content row.
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	v := plainView(m)
	tabbFrame(t, "settings", v, 100, 40, []int{2, 4}, 29, 37)
	if l := strings.Split(v, "\n")[3]; !strings.HasPrefix(l, "│Settings ") || !strings.HasSuffix(l, " ●│") {
		t.Errorf("settings header: %q", l)
	}
}

func TestTabbRenderTabBox(t *testing.T) {
	for _, c := range []struct {
		name                           string
		w, h                           int
		top, left, right, list, footer string
		want                           string
	}{
		{"content shorter than the box, two footer lines", 12, 12, "Tabs", "left", "right", "a\nb\n", "f1\nf2", `┌──────────┐
│Tabs      │
├──────────┤
│left right│
├──────────┤
│a         │
│b         │
│          │
├──────────┤
│f1        │
│f2        │
└──────────┘`},
		{"content taller than the box", 8, 8, "T", "L", "", "1\n2\n3\n4\n5", "", `┌──────┐
│T     │
├──────┤
│L     │
├──────┤
│1     │
│2     │
└──────┘`},
		{"content fills the box, a blank footer adds no line", 8, 8, "T", "", "R", "1\n2\n", " \n", `┌──────┐
│T     │
├──────┤
│     R│
├──────┤
│1     │
│2     │
└──────┘`},
		{"lines wider than the box are cut, blank lines kept", 6, 11, "T", "", "", "abcdefgh\n\n  x", "footer-wide", `┌────┐
│T   │
├────┤
│    │
├────┤
│abcd│
│    │
│  x │
├────┤
│foot│
└────┘`},
		{"a header wider than the box cuts the left part", 8, 7, "Tabs too wide", "longleft", "R", "", "", `┌──────┐
│Tabs t│
├──────┤
│long R│
├──────┤
│      │
└──────┘`},
		{"a right part wider than the box takes the header", 8, 6, "T", "L", "status-wide", "", "", `┌──────┐
│T     │
├──────┤
│status│
├──────┤
└──────┘`},
		{"blank content gives blank rows", 6, 7, "T", "", "", "  \n\n", "", `┌────┐
│T   │
├────┤
│    │
├────┤
│    │
└────┘`},
		{"just room for the footer, no content rows", 6, 8, "T", "L", "", "a", "F", `┌────┐
│T   │
├────┤
│L   │
├────┤
├────┤
│F   │
└────┘`},
		{"a footer with a blank line", 6, 10, "T", "", "", "", "F\n\nG\n", `┌────┐
│T   │
├────┤
│    │
├────┤
├────┤
│F   │
│    │
│G   │
└────┘`},
		{"one column inside: the right part wins", 3, 9, "T", "L", "R", "a\nb\n", "F", "┌─┐\n│T│\n├─┤\n│R│\n├─┤\n│a│\n├─┤\n│F│\n└─┘"},
		{"no column inside", 2, 8, "T", "L", "R", "a\nb", "", "┌┐\n││\n├┤\n││\n├┤\n││\n││\n└┘"},
		{"one column: empty lines", 1, 8, "T", "L", "R", "a\nb", "", strings.Repeat("\n", 7)},
		{"one row: the top border only", 10, 1, "T", "L", "R", "a", "F", "┌────────┐"},
		{"two rows: the top border only", 10, 2, "T", "L", "R", "a", "F", "┌────────┐"},
		{"one cell", 1, 1, "T", "L", "R", "a", "F", ""},
		{"no width: the lines without a box", 0, 10, "T", "L", "R", "a\nb\n\n", "F", "T\nL\na\nb"},
		{"no height: the lines without a box", 10, 0, "T", "L", "R", "a\nb\n\n", "F", "T\nL\na\nb"},
		{"no size, no header, no content", -1, -1, "T", "", "", "", "", "T"},
	} {
		if got := renderTabBox(c.w, c.h, c.top, c.left, c.right, c.list, c.footer); got != c.want {
			t.Errorf("%s (%dx%d):\n got:\n%s\nwant:\n%s\n got %q", c.name, c.w, c.h, got, c.want, got)
		}
	}
}

func TestTabbBoxParts(t *testing.T) {
	for _, c := range []struct{ name, got, want string }{
		{"top 0", boxTop(0), ""},
		{"top 1", boxTop(1), ""},
		{"top 2", boxTop(2), "┌┐"},
		{"top 3", boxTop(3), "┌─┐"},
		{"sep 5", boxSep(5), "├───┤"},
		{"bottom 5", boxBottom(5), "└───┘"},
		{"focused top 4", focusedBoxTop(4), "┌──┐"},
		{"focused sep 4", focusedBoxSep(4), "├──┤"},
		{"focused bottom 4", focusedBoxBottom(4), "└──┘"},
		{"line -1", boxLine(-1, "x"), ""},
		{"line 1", boxLine(1, "x"), ""},
		{"line 2", boxLine(2, "x"), "││"},
		{"line 3", boxLine(3, "x"), "│x│"},
		{"line padded", boxLine(5, "x"), "│x  │"},
		{"line exact", boxLine(5, "abc"), "│abc│"},
		{"line cut", boxLine(5, "abcdef"), "│abc│"},
		{"line wide letters cut", boxLine(5, "日本語"), "│日 │"},
		{"line trailing newlines", boxLine(5, "ab\n\n"), "│ab │"},
		// A line that fits is not rendered again: lipgloss would turn its
		// tab, which counts 0 cells, into spaces.
		{"line fits with a tab", boxLine(4, "ab\t"), "│ab\t│"},
		{"focused line", focusedBoxLine(5, "x"), "│x  │"},

		{"pad no width", padVisible("ab\n", 0), "ab\n"},
		{"pad below 0", padVisible("abc", -1), "abc"},
		{"pad 1", padVisible("abc", 1), "a"},
		{"pad to width", padVisible("ab", 3), "ab "},
		{"pad exact", padVisible("ab", 2), "ab"},
		{"pad cut", padVisible("abcdef", 3), "abc"},
		{"pad trailing newline", padVisible("ab\n", 4), "ab  "},
		{"pad fits with a tab", padVisible("ab\t", 2), "ab\t"},

		{"tabs Hosts", renderTabsLine(0, mainTabs), "Hosts  Groups  Settings"},
		{"tabs none", renderTabsLine(-1, nil), ""},
		{"main box", renderMainTabBox(8, 6, 2, "L", "R", "a"), "┌──────┐\n│Hosts │\n├──────┤\n│L    R│\n├──────┤\n└──────┘"},
		{"breadcrumb box", renderBreadcrumbTabBox(8, 8, "a › b", "L", "", "x", "F"), "┌──────┐\n│a › b │\n├──────┤\n│L     │\n├──────┤\n├──────┤\n│F     │\n└──────┘"},

		{"title blank", focusedBoxTitleTop(10, "  "), "┌────────┐"},
		{"title 2", focusedBoxTitleTop(2, "T"), "┌┐"},
		{"title 1", focusedBoxTitleTop(1, "T"), ""},
		{"title trimmed", focusedBoxTitleTop(9, " T "), "┌ T ────┐"},
		{"title exact", focusedBoxTitleTop(5, "a"), "┌ a ┐"},
		{"title cut", focusedBoxTitleTop(8, "abcdefgh"), "┌ abc… ┐"},
		{"title cut to nothing", focusedBoxTitleTop(4, "abc"), "┌  ┐"},
		{"title wide letters cut", focusedBoxTitleTop(6, "日本語"), "┌ … ─┐"},
	} {
		if c.got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, c.got, c.want)
		}
	}
}

// TestTabbColour checks the style codes the tab box adds.
func TestTabbColour(t *testing.T) {
	if !rowpInColour(t) {
		return
	}
	t.Cleanup(func() { ApplyColorScheme("", "") })
	act, in := tabActiveStyle.Render, tabInactiveStyle.Render
	if act("x") == in("x") {
		t.Fatal("the active and inactive tab styles look the same")
	}
	if got, want := renderTabsLine(1, mainTabs), in("Hosts")+"  "+act("Groups")+"  "+in("Settings"); got != want {
		t.Errorf("tabs, Groups active:\n got %q\nwant %q", got, want)
	}

	// A theme with a background pads in it, and only where there is room.
	ApplyColorScheme("nord", "")
	pad := tabBoxPadStyle.Render
	if pad("") == "" {
		t.Fatal("nord pads with no background")
	}
	for _, c := range []struct{ name, got, want string }{
		{"pad exact", padVisible("ab", 2), "ab"},
		{"pad to width", padVisible("ab", 4), "ab" + pad("  ")},
	} {
		if c.got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, c.got, c.want)
		}
	}
}

// The box is never taller than its window: the terminal would push its top
// rows out. A window too short for the frame gets the top border only, as
// a window of 1 or 2 rows did.
func TestTabbFitsTheWindow(t *testing.T) {
	screens := map[string]func() tea.Model{
		"hosts":    func() tea.Model { return hostsScreen(nil) },
		"groups":   func() tea.Model { return groupsScreen(nil) },
		"settings": func() tea.Model { return newDefaultsFormModel(config.DefaultConfig().Defaults, false) },
	}
	for name, open := range screens {
		for _, w := range []int{40, 100} {
			for h := 1; h <= 12; h++ {
				m := open()
				m.Update(tea.WindowSizeMsg{Width: w, Height: h})
				if v := plainView(m); strings.Count(v, "\n")+1 > h {
					t.Errorf("%s %dx%d: %d lines:\n%s", name, w, h, strings.Count(v, "\n")+1, v)
				}
			}
		}
	}
	for _, c := range []struct {
		h      int
		footer string
	}{{3, ""}, {5, ""}, {3, "F"}, {7, "F"}, {8, "F\nG"}} {
		if got := renderTabBox(8, c.h, "T", "L", "R", "a", c.footer); got != "┌──────┐" {
			t.Errorf("8x%d, footer %q:\n got:\n%s\nwant the top border only", c.h, c.footer, got)
		}
	}
}
