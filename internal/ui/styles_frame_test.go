package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"
)

// stylFrame is renderFocusedFrame without colours.
func stylFrame(w, h int, title, right, body, footer string) string {
	return ansi.Strip(renderFocusedFrame(w, h, title, right, body, footer))
}

// stylLine is the first line of view that holds s.
func stylLine(t *testing.T, view, s string) string {
	t.Helper()
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, s) {
			return l
		}
	}
	t.Fatalf("no line holds %q:\n%s", s, view)
	return ""
}

// A modal frame: one column of padding inside the border, the header with
// its right part flush right, the body without its trailing blank lines,
// the footer, then blank rows to the full height.
func TestStylFocusedFrameLayout(t *testing.T) {
	for _, c := range []struct {
		name                       string
		w, h                       int
		title, right, body, footer string
		want                       string
	}{
		{"header, body, footer", 24, 7, "Title", "R", "body\n\n", "foot", `┌──────────────────────┐
│ Title              R │
│ body                 │
│ foot                 │
│                      │
│                      │
└──────────────────────┘`},
		{"a blank footer adds no line", 24, 5, "Title", "", "body", "  ", `┌──────────────────────┐
│ Title                │
│ body                 │
│                      │
└──────────────────────┘`},
		{"a right part wider than the frame takes the header", 7, 4, "Title", "RRRR", "b", "", `┌─────┐
│ RRR │
│ b   │
└─────┘`},
	} {
		if got := stylFrame(c.w, c.h, c.title, c.right, c.body, c.footer); got != c.want {
			t.Errorf("%s:\n got:\n%s\nwant:\n%s", c.name, got, c.want)
		}
	}
}

// Before a modal knows its size it draws its lines without a frame.
func TestStylModalBeforeItsSize(t *testing.T) {
	m := newCustomHostModel(0, screenGroups)
	want := "Connect\n" + ansi.Strip(m.input.View()) + "\n⏎ connect  Ctrl+a add to group  Esc cancel"
	if got := plainView(m); got != want {
		t.Fatalf("no size:\n got %q\nwant %q", got, want)
	}
	m.toast = toast{text: "host required", level: toastWarn}
	if got := plainView(m); got != want+"\nhost required" {
		t.Fatalf("no size, a toast:\n got %q\nwant %q", got, want+"\nhost required")
	}

	// Either side unknown: the header on one line, the body and footer
	// trimmed. Both sides known, however small: a frame.
	for _, s := range [][2]int{{0, 6}, {20, 0}, {-1, -1}} {
		if got, want := stylFrame(s[0], s[1], " Title ", " R ", "  body  \n", "foot"), "Title R\nbody\nfoot"; got != want {
			t.Errorf("%dx%d: got %q, want %q", s[0], s[1], got, want)
		}
	}
	for _, s := range [][2]int{{1, 5}, {20, 1}} {
		if got := stylFrame(s[0], s[1], "T", "", "b", ""); !strings.HasPrefix(got, "┌") || !strings.Contains(got, "│ T") {
			t.Errorf("%dx%d: no frame:\n%s", s[0], s[1], got)
		}
	}
}

func TestStylJoinHeader(t *testing.T) {
	for _, c := range []struct {
		width       int
		left, right string
		want        string
	}{
		{0, " a ", " b ", "a b"}, // width unknown: both, one space apart
		{0, "", "b", "b"},
		{0, "a", "", "a"},
		{1, "a", "b", "b"},
		{10, " ab ", " xyz ", "ab     xyz"},   // right part flush right
		{10, "abcdefgh", "xyz", "abcdef xyz"}, // left cut, one space kept
		{10, "", "xyz", "       xyz"},
		{5, "abc", "xyz", "a xyz"},
		{4, "abc", "xyz", " xyz"}, // room for the space only
		{3, "abc", "xyz", "xyz"},  // the right part fills it
		{3, "abc", "wxyz", "wxy"}, // the right part cut
		{10, "ab", "", "ab"},
	} {
		if got := joinHeader(c.width, c.left, c.right); got != c.want {
			t.Errorf("joinHeader(%d, %q, %q) = %q, want %q", c.width, c.left, c.right, got, c.want)
		}
	}
}

func TestStylFrameInnerSize(t *testing.T) {
	for _, c := range []struct{ w, h, innerW, innerH int }{
		{40, 10, 36, 8}, // the border takes 2 columns and 2 rows, the padding 2 columns
		{4, 2, 0, 0},
		{3, 1, 0, 0}, // never below 0
	} {
		if w, h := frameInnerSize(c.w, c.h); w != c.innerW || h != c.innerH {
			t.Errorf("frameInnerSize(%d, %d) = %d, %d, want %d, %d", c.w, c.h, w, h, c.innerW, c.innerH)
		}
	}
}

func TestStylFormScrollWindow(t *testing.T) {
	for _, c := range []struct{ total, visible, focus, start, end int }{
		{3, 5, 2, 0, 3}, // all lines fit
		{5, 5, 4, 0, 5},
		{10, 4, 0, 0, 4}, // at the top
		{10, 4, 1, 0, 4}, // near the top: the window starts at the first line
		{10, 4, 5, 3, 7}, // the focused line in the middle
		{10, 5, 5, 3, 8},
		{10, 4, 8, 6, 10},
		{10, 4, 9, 6, 10}, // near the end: the window ends at the last line
	} {
		if s, e := formScrollWindow(c.total, c.visible, c.focus); s != c.start || e != c.end {
			t.Errorf("formScrollWindow(%d, %d, %d) = %d, %d, want %d, %d", c.total, c.visible, c.focus, s, e, c.start, c.end)
		}
	}
}

func TestStylFormSection(t *testing.T) {
	for _, c := range []struct {
		label string
		width int
		want  string
	}{
		{" Net ", 12, "── Net ─────"},
		{"Network", 5, "── Network "}, // too narrow: no fill
	} {
		if got := ansi.Strip(formSection(c.label, c.width)); got != c.want {
			t.Errorf("formSection(%q, %d) = %q, want %q", c.label, c.width, got, c.want)
		}
	}
}

// stylKeepSpinner puts the spinner back as it was after the test.
func stylKeepSpinner(t *testing.T) {
	active, index, minEnd := spinnerActive, spinnerIndex, spinnerMinEnd
	t.Cleanup(func() { spinnerActive, spinnerIndex, spinnerMinEnd = active, index, minEnd })
}

// A reload shows the spinner from its first frame. A reload that ends in
// the spinner's minimum time leaves it on; after that time it goes off.
func TestStylReloadSpinner(t *testing.T) {
	stylKeepSpinner(t)
	done := knownHostsReloadMsg{res: hosts.LoadResult{Hosts: []string{"alpha", "beta"}}}

	m := hostsScreen(nil)
	on(m, "r")
	screenShows(t, m, []string{"⠋ reloading"})
	m.Update(done)
	screenShows(t, m, []string{"⠋ 2 hosts loaded"})

	on(m, "r")
	spinnerMinEnd = time.Now().Add(-time.Millisecond)
	m.Update(done)
	screenShows(t, m, []string{"2 hosts loaded"}, "⠋")
}

// The config and count badges keep their column when the cursor moves onto
// their row: the active row writes them as plain text of the same width.
func TestStylBadgesKeepTheirColumn(t *testing.T) {
	cursorOff := func(s string) string { return strings.Replace(s, "▸", " ", 1) }

	m := hostsScreen(func(o *Options) { o.Inventory.Hosts = []config.Host{{Host: "beta", User: "u"}} })
	off := stylLine(t, plainView(m), "beta")
	on(m, "j")
	if onRow := stylLine(t, plainView(m), "beta"); !strings.Contains(off, "◻ beta  ⚙ ") || cursorOff(onRow) != off {
		t.Fatalf("config badge moved:\n%s\n%s", off, onRow)
	}

	g := groupsScreen(nil) // db, empty, web
	off = stylLine(t, plainView(g), "web")
	on(g, "j", "j")
	if onRow := stylLine(t, plainView(g), "web"); !strings.Contains(off, "  web  1 ") || cursorOff(onRow) != off {
		t.Fatalf("count badge moved:\n%s\n%s", off, onRow)
	}
}

// The help box has a blank row above its title and two columns before it.
func TestStylHelpBoxPadding(t *testing.T) {
	lines := strings.Split(plainView(hostsHelp(60, 50)), "\n")
	for i, l := range lines {
		if !strings.Contains(l, "Hosts keybindings") {
			continue
		}
		if i < 2 || strings.Trim(lines[i-1], " │") != "" || !strings.Contains(lines[i-2], "┌") || !strings.Contains(l, "│  Hosts keybindings") {
			t.Fatalf("help title not padded:\n%s", strings.Join(lines, "\n"))
		}
		return
	}
	t.Fatal("no help title")
}
