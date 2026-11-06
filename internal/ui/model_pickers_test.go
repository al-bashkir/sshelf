package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// sendKeys sends keys and drops their commands: a key typed into a text
// input returns a cursor blink that sleeps.
func sendKeys(m tea.Model, ks ...string) {
	for _, k := range ks {
		m.Update(hk(k))
	}
}

// hostPicker opens the host picker on alpha, beta and gamma in a 60x20 modal.
func hostPicker() *hostPickerModel {
	m := newHostPickerModel(Options{Config: config.DefaultConfig(), Hosts: []string{"alpha", "beta", "gamma"}})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	return m
}

// groupPicker opens the group picker on groups web, db and Alpha (indexes
// 0, 1, 2), adding hosts, in a 60x20 modal.
func groupPicker(hosts ...string) *groupPickerModel {
	inv := config.Inventory{Groups: []config.Group{{Name: "web"}, {Name: "db"}, {Name: "Alpha"}}}
	m := newGroupPickerModel(Options{Config: config.DefaultConfig(), Inventory: inv})
	m.hosts = hosts
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	return m
}

type pickerCase struct {
	name      string
	new       func() tea.Model
	cancelled func(tea.Msg) bool
	rows      func(tea.Model) []string
	focus     func(tea.Model) focusState
	query     func(tea.Model) string
}

func pickerCases() []pickerCase {
	return []pickerCase{
		{
			name:      "host",
			new:       func() tea.Model { return hostPicker() },
			cancelled: func(msg tea.Msg) bool { _, ok := msg.(hostPickerCancelMsg); return ok },
			rows:      func(m tea.Model) []string { return rowHosts(m.(*hostPickerModel).list.Items()) },
			focus:     func(m tea.Model) focusState { return m.(*hostPickerModel).focus },
			query:     func(m tea.Model) string { return m.(*hostPickerModel).search.Value() },
		},
		{
			name:      "group",
			new:       func() tea.Model { return groupPicker() },
			cancelled: func(msg tea.Msg) bool { _, ok := msg.(groupPickerCancelMsg); return ok },
			rows:      func(m tea.Model) []string { return groupPickRows(m.(*groupPickerModel)) },
			focus:     func(m tea.Model) focusState { return m.(*groupPickerModel).focus },
			query:     func(m tea.Model) string { return m.(*groupPickerModel).search.Value() },
		},
	}
}

func groupPickRows(m *groupPickerModel) []string {
	var out []string
	for _, it := range m.list.Items() {
		out = append(out, it.(groupPickRow).name)
	}
	return out
}

func TestPickersFitTheWindow(t *testing.T) {
	for _, c := range pickerCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			lines := strings.Split(m.View(), "\n")
			if len(lines) != 20 {
				t.Fatalf("%d lines, want 20", len(lines))
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w != 60 {
					t.Fatalf("line %d is %d wide, want 60", i, w)
				}
			}
		})
	}
	// The list gets the frame less five rows and the adding lines; the
	// search box the frame width less its prompt, at least 10.
	h := hostPicker()
	if h.list.Width() != 56 || h.list.Height() != 13 || h.search.Width != 54 {
		t.Fatalf("host picker list %dx%d search %d, want 56x13 and 54", h.list.Width(), h.list.Height(), h.search.Width)
	}
	g := groupPicker("alpha", "beta")
	if g.list.Width() != 56 || g.list.Height() != 12 || !strings.Contains(plainView(g), "adding 2: alpha, beta") {
		t.Fatalf("group picker list %dx%d, want 56x12 under the adding line:\n%s", g.list.Width(), g.list.Height(), plainView(g))
	}
	h.Update(tea.WindowSizeMsg{Width: 12, Height: 20})
	if h.search.Width != 10 {
		t.Fatalf("narrow search width %d, want 10", h.search.Width)
	}
}

func TestPickersHelp(t *testing.T) {
	for _, c := range pickerCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			sendKeys(m, "?")
			if !strings.Contains(plainView(m), "Navigation") {
				t.Fatalf("? did not show help:\n%s", plainView(m))
			}
			// Help takes every key; only ? and Esc close it.
			for _, k := range []string{"enter", "/", "j"} {
				if got := keys(m, k); got != nil || c.focus(m) != focusList || !strings.Contains(plainView(m), "Navigation") {
					t.Fatalf("%s with help open: emitted %T, focus %v", k, got, c.focus(m))
				}
			}
			for _, k := range []string{"?", "esc"} {
				if got := keys(m, k); got != nil || strings.Contains(plainView(m), "Navigation") {
					t.Fatalf("%s did not just close help (emitted %T)", k, got)
				}
				sendKeys(m, "?")
			}
		})
	}
}

func TestPickersEscClearsThenCancels(t *testing.T) {
	for _, c := range pickerCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			all := c.rows(m)
			sendKeys(m, "/", "a", "l")
			if c.focus(m) != focusSearch || c.query(m) != "al" || len(c.rows(m)) >= len(all) {
				t.Fatalf("typing did not filter: focus %v query %q rows %q", c.focus(m), c.query(m), c.rows(m))
			}
			// Esc clears the query and keeps the search box focused...
			if got := keys(m, "esc"); got != nil || c.query(m) != "" || c.focus(m) != focusSearch || !slices.Equal(c.rows(m), all) {
				t.Fatalf("1st Esc: emitted %T focus %v query %q rows %q", got, c.focus(m), c.query(m), c.rows(m))
			}
			// ...then leaves it...
			if got := keys(m, "esc"); got != nil || c.focus(m) != focusList {
				t.Fatalf("2nd Esc: emitted %T focus %v", got, c.focus(m))
			}
			// ...then cancels.
			if !c.cancelled(keys(m, "esc")) {
				t.Fatal("3rd Esc did not cancel")
			}

			// With the list focused, Esc clears the query first too.
			m = c.new()
			sendKeys(m, "ctrl+f", "a", "l", "tab")
			if got := keys(m, "esc"); got != nil || c.query(m) != "" || c.focus(m) != focusList {
				t.Fatalf("Esc in the list: emitted %T focus %v query %q", got, c.focus(m), c.query(m))
			}
			if !c.cancelled(keys(m, "esc")) {
				t.Fatal("Esc with nothing to clear did not cancel")
			}
		})
	}
}

func TestPickersFocusKeys(t *testing.T) {
	for _, c := range pickerCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			for _, s := range []struct {
				key  string
				want focusState
			}{{"tab", focusSearch}, {"tab", focusList}, {"/", focusSearch}, {"/", focusSearch}, {"enter", focusList}, {"ctrl+f", focusSearch}} {
				if got := keys(m, s.key); got != nil || c.focus(m) != s.want {
					t.Fatalf("%s: emitted %T focus %v, want %v", s.key, got, c.focus(m), s.want)
				}
			}
			if c.query(m) != "" {
				t.Fatalf("focus keys typed %q", c.query(m))
			}
		})
	}
}

func TestHostPickerEnterAddsCurrentOrSelected(t *testing.T) {
	done := func(msg tea.Msg) []string {
		d, ok := msg.(hostPickerDoneMsg)
		if !ok {
			t.Fatalf("emitted %T, want done", msg)
		}
		return d.hosts
	}
	m := hostPicker()
	if got := done(keys(m, "j", "enter")); !slices.Equal(got, []string{"beta"}) {
		t.Fatalf("Enter added %q, want the current host beta", got)
	}
	sendKeys(m, "space", "j", " ")
	if got := done(keys(m, "enter")); !slices.Equal(got, []string{"beta", "gamma"}) {
		t.Fatalf("Enter added %q, want the selected beta and gamma", got)
	}
	sendKeys(m, "ctrl+d")
	if got := done(keys(m, "enter")); !slices.Equal(got, []string{"gamma"}) {
		t.Fatalf("after Ctrl+D Enter added %q, want only the current gamma", got)
	}
	// Ctrl+A selects the rows the search shows.
	sendKeys(m, "/", "a", "enter", "ctrl+a", "esc")
	if got := done(keys(m, "enter")); !slices.Equal(got, []string{"alpha", "beta", "gamma"}) {
		t.Fatalf("Ctrl+A on 'a' added %q", got)
	}
	// In the search box, Space, Ctrl+A and Ctrl+D do not select.
	m = hostPicker()
	sendKeys(m, "/", " ", "ctrl+a", "ctrl+d", "enter")
	if got := done(keys(m, "enter")); !slices.Equal(got, []string{"alpha"}) {
		t.Fatalf("selection keys in the search box: Enter added %q, want alpha", got)
	}
}

func TestGroupPickerEnterPicksTheCurrentGroup(t *testing.T) {
	picked := func(m *groupPickerModel, ks ...string) int {
		t.Helper()
		d, ok := keys(m, ks...).(groupPickerDoneMsg)
		if !ok {
			t.Fatal("Enter did not pick a group")
		}
		return d.groupIndex
	}
	m := groupPicker()
	if rows := groupPickRows(m); !slices.Equal(rows, []string{"Alpha", "db", "web"}) {
		t.Fatalf("rows %q, want sorted without case", rows)
	}
	if got := picked(m, "enter"); got != 2 {
		t.Fatalf("picked index %d, want 2 (Alpha)", got)
	}
	if got := picked(m, "space", "j", "enter"); got != 1 {
		t.Fatalf("picked index %d, want 1 (db); Space must not select", got)
	}
	// A subgroup shows its path: two groups can share a name.
	inv := config.Inventory{Groups: []config.Group{{Name: "prod"}, {Name: "web", Parent: "prod"}, {Name: "web"}}}
	nested := newGroupPickerModel(Options{Config: config.DefaultConfig(), Inventory: inv})
	if rows := groupPickRows(nested); !slices.Equal(rows, []string{"prod", "prod/web", "web"}) {
		t.Fatalf("rows %q, want paths", rows)
	}
	// No group to pick: Enter does nothing.
	empty := newGroupPickerModel(Options{Config: config.DefaultConfig()})
	if got := keys(empty, "enter"); got != nil {
		t.Fatalf("Enter with no groups emitted %T", got)
	}
}

func TestGroupPickerSearch(t *testing.T) {
	m := groupPicker()
	sendKeys(m, "/", "w", "e")
	if rows := groupPickRows(m); !slices.Equal(rows, []string{"web"}) {
		t.Fatalf("rows %q, want web", rows)
	}
	if got := m.list.Items()[0].(groupPickRow).matchedIndexes; !slices.Equal(got, []int{0, 1}) {
		t.Fatalf("matched %v, want [0 1]", got)
	}
	if d, ok := keys(m, "enter", "enter").(groupPickerDoneMsg); !ok || d.groupIndex != 0 {
		t.Fatalf("picked %#v, want web (0)", d)
	}
	// Clearing the search shows every group again, unmarked, from the top.
	sendKeys(m, "j", "esc")
	for _, it := range m.list.Items() {
		if it.(groupPickRow).matchedIndexes != nil {
			t.Fatal("a row kept its match marks")
		}
	}
	if rows := groupPickRows(m); !slices.Equal(rows, []string{"Alpha", "db", "web"}) || m.list.Index() != 0 {
		t.Fatalf("rows %q cursor %d, want all from the top", rows, m.list.Index())
	}
	// Spaces around the query do not count; a query that matches nothing
	// leaves nothing to pick.
	sendKeys(m, "/", " ", "d", "b", " ")
	if rows := groupPickRows(m); !slices.Equal(rows, []string{"db"}) {
		t.Fatalf("rows %q, want db", rows)
	}
	sendKeys(m, "esc", "z", "z")
	if rows := groupPickRows(m); len(rows) != 0 {
		t.Fatalf("rows %q, want none", rows)
	}
	if got := keys(m, "tab", "enter"); got != nil {
		t.Fatalf("Enter with no match emitted %T", got)
	}
}

func TestGroupPickerAddingLines(t *testing.T) {
	hostsN := func(n int, name string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = name + string(rune('1'+i))
		}
		return out
	}
	for _, c := range []struct {
		name  string
		hosts []string
		width int
		want  []string
	}{
		{"none", nil, 40, nil},
		{"one", []string{"alpha"}, 40, []string{"adding: alpha"}},
		{"a few", []string{"a", "b", "c"}, 40, []string{"adding 3: a, b, c"}},
		{"too narrow for names", []string{"alpha", "beta"}, 12, []string{"adding 2"}},
		{"a long name is cut to fit", []string{"a-very-long-hostname-here"}, 20, []string{"adding: a-very-long…"}},
		{"whole names wrap to the second line", hostsN(4, "alpha-"), 30,
			[]string{"adding 4: alpha-1, alpha-2", "alpha-3, alpha-4"}},
		{"the rest as +N more", hostsN(8, "server-0000"), 40,
			[]string{"adding 8: server-00001, server-00002", "server-00003, server-00004, +4 more"}},
		{"+N when +N more does not fit", hostsN(9, "h-0000000"), 40,
			[]string{"adding 9: h-00000001, h-00000002", "h-00000003, h-00000004, h-00000005, +4"}},
		{"a long name wraps whole", []string{"alpha-1", "very-long-host-name-x"}, 30,
			[]string{"adding 2: alpha-1", "very-long-host-name-x"}},
		{"a name too long for the second line is cut", []string{"alpha-1", "a-really-very-long-host-name-that-goes-on"}, 30,
			[]string{"adding 2: alpha-1", "a-really-very-long-host-name-…"}},
	} {
		m := groupPicker(c.hosts...)
		var got []string
		for _, l := range m.addingLines(c.width) {
			got = append(got, ansi.Strip(l))
			if w := lipgloss.Width(l); w > c.width {
				t.Errorf("%s: line %q is %d wide, more than %d", c.name, ansi.Strip(l), w, c.width)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// Hidden hosts can still be added to a group, so the picker lists them,
// unmarked.
func TestHostPickerListsHiddenHosts(t *testing.T) {
	m := newHostPickerModel(Options{Config: config.DefaultConfig(), Hosts: []string{"alpha", "beta"},
		Inventory: config.Inventory{Hosts: []config.Host{{Host: "beta", Hidden: true}}}})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	screenShows(t, m, []string{"alpha", "beta"}, "⊘")
	sendKeys(m, "/", "b")
	screenShows(t, m, []string{"beta"}, "⊘")
}

// A search moves the cursor to the first match.
func TestGroupPickerSearchMovesToTheTop(t *testing.T) {
	m := groupPicker()
	sendKeys(m, "j", "j", "/", "d", "b")
	if d, ok := keys(m, "enter", "enter").(groupPickerDoneMsg); !ok || d.groupIndex != 1 {
		t.Fatalf("picked %#v, want db (1)", d)
	}
}

func TestGroupPickerAddingLinesAtTheLimits(t *testing.T) {
	a, b, c := strings.Repeat("a", 28), strings.Repeat("b", 15), strings.Repeat("c", 14)
	for _, tc := range []struct {
		name  string
		hosts []string
		width int
		want  []string
	}{
		{"just wide enough for names", []string{"alpha", "beta"}, 13, []string{"adding 2: al…", "beta"}},
		{"a name that fills the line exactly stays", []string{"alpha-1", "abcdefghijk"}, 30,
			[]string{"adding 2: alpha-1, abcdefghijk"}},
		{"+N more that fills the line exactly", []string{a, b, c, strings.Repeat("d", 10), "eee"}, 40,
			[]string{"adding 5: " + a, b + ", " + c + ", +2 more"}},
		{"room for +N is kept from the second line's own width", []string{
			strings.Repeat("a", 25), strings.Repeat("b", 13), strings.Repeat("c", 13), "dddd", "e"}, 40,
			[]string{"adding 5: " + strings.Repeat("a", 25), strings.Repeat("b", 13) + ", " + strings.Repeat("c", 13) + ", dddd, e"}},
	} {
		m := groupPicker(tc.hosts...)
		var got []string
		for _, l := range m.addingLines(tc.width) {
			got = append(got, ansi.Strip(l))
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPickerStatusLine(t *testing.T) {
	screenShows(t, hostPicker(), []string{"sel:0  search"}, "pg:", "search:")

	hs := make([]string, 30)
	for i := range hs {
		hs[i] = fmt.Sprintf("h%02d", i)
	}
	m := newHostPickerModel(Options{Config: config.DefaultConfig(), Hosts: hs})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	screenShows(t, m, []string{"pg:1/"})
	m.toast = toast{text: "boom", level: toastErr}
	screenShows(t, m, []string{"boom"})

	sendKeys(m, "/", "h", "0")
	screenShows(t, m, []string{"search: h0"})
	x38 := strings.Repeat("x", 38)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(x38)})
	screenShows(t, m, []string{"search: h0" + x38 + "  "}, x38+"...") // 40 characters fit
	sendKeys(m, "x")
	screenShows(t, m, []string{"search: h0" + x38 + "..."})
}

func TestGroupPickerRowsSitOnConsecutiveLines(t *testing.T) {
	inv := config.Inventory{Groups: []config.Group{{Name: "web"}, {Name: "Web"}, {Name: "db"}}}
	m := newGroupPickerModel(Options{Config: config.DefaultConfig(), Inventory: inv})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	lines := strings.Split(plainView(m), "\n")
	row := func(name string) int {
		return slices.IndexFunc(lines, func(l string) bool { return strings.HasSuffix(strings.TrimRight(l, " │"), " "+name) })
	}
	// Sorted without case; equal names keep the inventory order.
	if db, web, Web := row("db"), row("web"), row("Web"); db < 0 || web != db+1 || Web != web+1 {
		t.Fatalf("rows db %d, web %d, Web %d, want consecutive in that order:\n%s", db, web, Web, plainView(m))
	}
	if !strings.Contains(lines[row("db")], "▸ db") || strings.Contains(lines[row("web")], "▸") {
		t.Fatalf("the cursor is not on db alone:\n%s", plainView(m))
	}
}
