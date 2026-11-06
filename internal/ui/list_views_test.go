package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// screenShows fails unless the plain view shows every want and none of
// lacks.
func screenShows(t *testing.T, m tea.Model, want []string, lacks ...string) {
	t.Helper()
	v := plainView(m)
	for _, w := range want {
		if !strings.Contains(v, w) {
			t.Fatalf("view lacks %q:\n%s", w, v)
		}
	}
	for _, l := range lacks {
		if strings.Contains(v, l) {
			t.Fatalf("view shows %q:\n%s", l, v)
		}
	}
}

// footerCase is the footer a list screen shows at one window size.
type footerCase struct {
	w, h  int
	want  []string
	lacks []string
}

// footerCases checks compact (narrow), wide and wide+second (tall
// enough) footers at the size limits.
func footerCases(compact, wide, second string) []footerCase {
	return []footerCase{
		{100, 30, []string{wide, second}, []string{compact}},
		{100, 20, []string{wide, second}, []string{compact}},
		{100, 19, []string{wide}, []string{second, compact}},
		{60, 30, []string{wide, second}, []string{compact}},
		{59, 30, []string{compact}, []string{wide, second}},
	}
}

func checkFooters(t *testing.T, m tea.Model, cases []footerCase) {
	t.Helper()
	for _, c := range cases {
		m.Update(tea.WindowSizeMsg{Width: c.w, Height: c.h})
		screenShows(t, m, c.want, c.lacks...)
	}
}

func TestHostsViewStatus(t *testing.T) {
	m := hostsScreen(nil)
	screenShows(t, m, []string{"● 3 hosts"}, "selected", "0 hidden", "load warnings")
	on(m, "/", "a", "l")
	screenShows(t, m, []string{"● 1 / 3 hosts"})
	on(m, "esc", "esc", "space")
	screenShows(t, m, []string{"●   1 selected  3 hosts"})

	m = hostsScreen(func(o *Options) { o.SkippedLines = 1 })
	screenShows(t, m, []string{"● load warnings 3 hosts"})
	m = hostsScreen(func(o *Options) { o.LoadErrors = []hosts.PathError{{Path: "x"}} })
	screenShows(t, m, []string{"● load warnings 3 hosts"})

	hidden := func(o *Options) { o.Inventory.Hosts = []config.Host{{Host: "beta", Hidden: true}} }
	m = hostsScreen(hidden)
	screenShows(t, m, []string{"● 2 hosts  1 hidden"}, "1 showed")
	on(m, "/", "a")
	screenShows(t, m, []string{"● 2 / 2 hosts  1 hidden"})
	on(m, "esc", "esc", "H")
	screenShows(t, m, []string{"● 3 hosts  1 showed"}, "1 hidden")

	// A toast, or the reload spinner, takes the place of the counts.
	m = hostsScreen(nil)
	m.Update(toastMsg{text: "copied", level: toastInfo})
	screenShows(t, m, []string{"copied"}, "3 hosts")
	stylKeepSpinner(t)
	spinnerStart()
	screenShows(t, m, []string{"copied"}, "3 hosts")
	m.toast = toast{}
	screenShows(t, m, nil, "3 hosts", "copied")
}

func TestHostsViewFooter(t *testing.T) {
	m := hostsScreen(nil)
	checkFooters(t, m, footerCases("↵ connect  ␣ select  ? help",
		"↵ connect  O pane  ·  ␣ select", "e config  Ctrl+o cmd  a add"))
	on(m, "space")
	checkFooters(t, m, footerCases("o panes  ␣ clear  ? help",
		"o panes  ·  Ctrl+o cmd", "e config  r reload"))
}

func TestHostsViewEmptyStates(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Hosts = nil })
	screenShows(t, m, []string{"No hosts found.", "c          custom host", "r          reload known_hosts", "Alt+3      open settings"},
		"loading is off")
	m = hostsScreen(func(o *Options) { o.Hosts = nil; o.Config.Defaults.LoadKnownHosts = false })
	screenShows(t, m, []string{"No hosts found.", "c          custom host", "Alt+3      open settings", "(known_hosts loading is off)"},
		"reload known_hosts")
	m = hostsScreen(func(o *Options) {
		o.Hosts = []string{"beta"}
		o.Inventory.Hosts = []config.Host{{Host: "beta", Hidden: true}}
	})
	screenShows(t, m, []string{"No hosts."}, "No hosts found.")
	m = hostsScreen(nil)
	on(m, "/", "z", "z")
	screenShows(t, m, []string{`No matches for "zz"`, "Esc to clear search"})
}

func TestHostsViewOverlays(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "ctrl+o")
	screenShows(t, m, []string{"Connect and run a remote command (keeps session open)."})
	m = hostsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	on(m, "q")
	if m.View() != renderQuitConfirm(100, 30) {
		t.Fatalf("q did not show the quit question:\n%s", plainView(m))
	}
}

// nestedGroups adds top over leaf (l1) and web over top, so the tree is
// db, empty, web > top > leaf.
func nestedGroups(o *Options) {
	o.Inventory.Groups[0].Hosts = []string{"w1"}
	o.Inventory.Groups = append(o.Inventory.Groups,
		config.Group{Name: "top", Parent: "web"}, config.Group{Name: "leaf", Parent: "web/top", Hosts: []string{"l1", "w1"}})
}

func TestGroupsViewShowsTheTree(t *testing.T) {
	m := groupsScreen(nestedGroups)
	if got, want := shownGroups(m), []string{"db", "empty", "web", "top", "leaf"}; !slices.Equal(got, want) {
		t.Fatalf("rows %v, want %v", got, want)
	}
	// Subgroups sit two cells further in per level; a count has the hosts
	// of every subgroup, each once.
	screenShows(t, m, []string{"│  web  2 ", "│    top  2 ", "│      leaf  2 "})

	on(m, "/", "l", "e", "a", "f")
	if got := shownGroups(m); !slices.Equal(got, []string{"leaf"}) {
		t.Fatalf("rows %v, want [leaf]", got)
	}
	screenShows(t, m, []string{"│▸ leaf  2 "}, "▸     leaf") // a search lists the matches flat
}

func TestGroupsView(t *testing.T) {
	m := groupsScreen(nil)
	screenShows(t, m, []string{"●   3 groups"})
	m.Update(toastMsg{text: "copied", level: toastInfo})
	screenShows(t, m, []string{"copied"}, "3 groups")

	m = groupsScreen(nil)
	checkFooters(t, m, footerCases("↵ open  C connect  ? help",
		"↵ open  C connect  ·  o panes", "e edit  d delete  y copy"))

	// The delete question keeps the count, drops the footer.
	m = groupsScreen(nil)
	on(m, "d")
	screenShows(t, m, []string{"Delete group?", `Delete "db" (2)?`, "●   3 groups"}, "C connect")
	m = groupsScreen(nil)
	on(m, "j", "d")
	screenShows(t, m, []string{`Delete "empty" (0)?`})
	m = groupsScreen(nil)
	on(m, "j", "j", "d") // web, the first group in the inventory
	screenShows(t, m, []string{`Delete "web" (1)?`})
	questionCentered(t, m, "Delete group?", 0)

	m = groupsScreen(func(o *Options) { o.Inventory.Groups = nil })
	screenShows(t, m, []string{"No groups yet.", "n — create a new group"})
	m = groupsScreen(nil)
	on(m, "/", "z", "z")
	screenShows(t, m, []string{`No matches for "zz"`})

	m = groupsScreen(nil)
	on(m, "ctrl+o")
	screenShows(t, m, []string{"Connect and run a remote command for all hosts (keeps sessions open)."})
	m = groupsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	on(m, "q")
	if m.View() != renderQuitConfirm(100, 30) {
		t.Fatalf("q did not show the quit question:\n%s", plainView(m))
	}
}

func TestGroupHostsView(t *testing.T) {
	m := groupHostsScreen(0, nil)
	screenShows(t, m, []string{"Groups > prod", "● 3 hosts"}, "selected")
	on(m, "/", "a", "l")
	screenShows(t, m, []string{"● 1 / 3 hosts"})
	on(m, "esc", "esc", "space")
	screenShows(t, m, []string{"● 3 hosts    1 selected"})
	m.Update(toastMsg{text: "copied", level: toastInfo})
	screenShows(t, m, []string{"copied"}, "3 hosts")

	m = groupHostsScreen(0, nil)
	checkFooters(t, m, footerCases("↵ connect  ␣ select  esc back  ? help",
		"↵ connect  O pane  ·  ␣ select", "e config  c custom  d remove"))

	// The remove question keeps the crumb, drops the counts and the footer.
	m = groupHostsScreen(0, nil)
	on(m, "d")
	screenShows(t, m, []string{"Groups > prod", "Remove hosts?", "from prod"}, "3 hosts", "O pane")
	questionCentered(t, m, "Remove hosts?", -1) // a row high: it leaves room for a footer it lacks

	screenShows(t, groupHostsScreen(1, nil), []string{"Groups > empty", "No hosts in this group.", "a — add hosts"})
	m = groupHostsScreen(0, nil)
	on(m, "/", "z", "z")
	screenShows(t, m, []string{`No matches for "zz"`})

	m = groupHostsScreen(0, nil)
	on(m, "ctrl+o")
	screenShows(t, m, []string{"Connect and run a remote command (keeps session open)."})
}

// The connect question fits a narrow window, on every list screen.
func TestListViewsConnectQuestion(t *testing.T) {
	for _, c := range []struct {
		name string
		m    tea.Model
		keys []string
		want string
	}{
		{"hosts", hostsScreen(func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 2 }), []string{"ctrl+a", "enter"}, "Connect 3 hosts?"},
		{"groups", groupsScreen(func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 1 }), []string{"C"}, "Connect 2 hosts?"},
		{"group hosts", groupHostsScreen(0, func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 2 }), []string{"ctrl+a", "enter"}, "Connect 3 hosts?"},
	} {
		on(c.m, c.keys...)
		screenShows(t, c.m, []string{c.want})
		c.m.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
		for i, l := range strings.Split(c.m.View(), "\n") {
			if w := lipgloss.Width(l); w > 40 {
				t.Fatalf("%s: line %d is %d wide in a 40-wide window", c.name, i, w)
			}
		}
	}
}

// questionCentered fails unless the box titled title sits in the middle of
// the list area (rows 5 to the last but one) at offset rows from the exact
// middle (negative is higher), and across the width give or take a column.
func questionCentered(t *testing.T, m tea.Model, title string, offset int) {
	t.Helper()
	lines := strings.Split(plainView(m), "\n")
	top := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, title) })
	if top < 0 {
		t.Fatalf("no %q box:\n%s", title, plainView(m))
	}
	bottom := top + slices.IndexFunc(lines[top:], func(l string) bool {
		return strings.Contains(l, "└") && !strings.HasPrefix(l, "└")
	})
	if d := (top - 5) - (len(lines) - 2 - bottom); d/2 != offset {
		t.Fatalf("%q box rows %d-%d: offset %d, want %d:\n%s", title, top, bottom, d/2, offset, plainView(m))
	}
	row := []rune(lines[top])
	left, right := slices.Index(row, '┌')-1, len(row)-2-slices.Index(row, '┐')
	if d := left - right; slices.Index(row, '┐') < 0 || d < -1 || d > 1 {
		t.Fatalf("%q box is off the middle by %d columns:\n%s", title, d, plainView(m))
	}
}

func TestEmptyListMessageIsCentered(t *testing.T) {
	lines := strings.Split(plainView(hostsScreen(func(o *Options) { o.Hosts = nil })), "\n")
	has := func(s string) func(string) bool { return func(l string) bool { return strings.Contains(l, s) } }
	dots := slices.IndexFunc(lines, has("·  ·  ·"))
	last := slices.IndexFunc(lines, has("open settings"))
	top := 4                 // the rule under the search box
	bottom := len(lines) - 4 // the rule over the footer
	if above, below := dots-top-1, bottom-last-1; above != below {
		t.Fatalf("%d rows above the message, %d below:\n%s", above, below, strings.Join(lines, "\n"))
	}
	row := []rune(lines[dots])
	left := slices.Index(row, '·') - 1
	right := len(row) - 1 - (slices.Index(row, '·') + len([]rune("·  ·  ·")))
	if left != 45 || right != 46 { // 98 columns less 7
		t.Fatalf("dots at %d columns from the left, %d from the right, want 45 and 46", left, right)
	}
}

func TestGroupsRowsSitOnConsecutiveLines(t *testing.T) {
	m := groupsScreen(func(o *Options) {
		o.Inventory.Groups = []config.Group{{Name: "web"}, {Name: "Web"}, {Name: "db"}}
	})
	lines := strings.Split(plainView(m), "\n")
	row := func(name string) int {
		return slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, " "+name+" ") })
	}
	// Sorted without case; equal names keep the inventory order.
	if db, web, Web := row("db"), row("web"), row("Web"); web != db+1 || Web != web+1 {
		t.Fatalf("rows db %d, web %d, Web %d, want consecutive in that order:\n%s", db, web, Web, plainView(m))
	}
	if !strings.Contains(lines[row("db")], "▸ db") || strings.Contains(lines[row("web")], "▸") {
		t.Fatalf("the cursor is not on db alone:\n%s", plainView(m))
	}
}
