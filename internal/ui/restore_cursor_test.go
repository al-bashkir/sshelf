package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/al-bashkir/sshelf/internal/config"
)

type strItem string

func (s strItem) FilterValue() string { return string(s) }

func TestRestoreCursor(t *testing.T) {
	all := []string{"a", "b", "c", "d", "e"}
	tc := []struct {
		name    string
		visible []string
		prev    string
		want    int
	}{
		{"prev still visible", []string{"c", "a", "e"}, "a", 1},
		{"prev hidden: nearest later one", []string{"e", "b", "d"}, "c", 2},
		{"prev hidden, none later: nearest earlier one", []string{"a", "c"}, "d", 1},
		{"prev unknown: first row", []string{"c", "b"}, "z", 0},
		{"first of all hidden: nearest later one", []string{"c", "b"}, "a", 1},
		{"no prev: first row", []string{"b", "c"}, "", 0},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			items := make([]list.Item, len(c.visible))
			for i, v := range c.visible {
				items[i] = strItem(v)
			}
			l := list.New(items, list.NewDefaultDelegate(), 80, 100)
			l.Select(len(items) - 1) // so a move to row 0 shows
			restoreCursor(&l, all, c.visible, c.prev)
			if got := l.Index(); got != c.want {
				t.Fatalf("index=%d want %d", got, c.want)
			}
		})
	}

	l := list.New(nil, list.NewDefaultDelegate(), 80, 100)
	restoreCursor(&l, all, nil, "a") // no rows: nothing to select, no panic
}

func groupsModelWith(names ...string) *groupsModel {
	var inv config.Inventory
	for _, n := range names {
		inv.Groups = append(inv.Groups, config.Group{Name: n})
	}
	m := newGroupsModel(Options{Config: config.DefaultConfig(), Inventory: inv})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	return m
}

func selectGroup(t *testing.T, m *groupsModel, name string) {
	t.Helper()
	for i, it := range m.list.Items() {
		if it.(groupRow).name == name {
			m.list.Select(i)
			return
		}
	}
	t.Fatalf("group %q not in list", name)
}

func selectedGroup(m *groupsModel) string {
	row, _ := m.list.SelectedItem().(groupRow)
	return row.name
}

func TestGroupsFilterKeepsSelection(t *testing.T) {
	m := groupsModelWith("date", "banana", "apple", "cherry") // rows sort by name

	selectGroup(t, m, "cherry")
	m.applyFilter(" e ") // apple, cherry, date
	if got := selectedGroup(m); got != "cherry" || len(m.list.Items()) != 3 {
		t.Fatalf("selected %q of %d rows, want cherry of 3", got, len(m.list.Items()))
	}

	m.applyFilter("")
	if got := selectedGroup(m); got != "cherry" || len(m.list.Items()) != 4 {
		t.Fatalf("empty query: selected %q of %d rows, want cherry of 4", got, len(m.list.Items()))
	}

	selectGroup(t, m, "banana")
	m.applyFilter("e") // banana has no "e": the next row in name order wins
	if got := selectedGroup(m); got != "cherry" {
		t.Fatalf("selected %q, want cherry", got)
	}

	selectGroup(t, m, "date")
	m.applyFilter("ban") // one row left, before the old selection
	if got := selectedGroup(m); got != "banana" {
		t.Fatalf("selected %q, want banana", got)
	}

	m.applyFilter("zzz")
	if n := len(m.list.Items()); n != 0 {
		t.Fatalf("%d rows for no match, want 0", n)
	}
}

func TestGroupsRefreshKeepsSelectionAcrossRename(t *testing.T) {
	m := groupsModelWith("banana", "apple", "cherry") // banana is inventory index 0
	selectGroup(t, m, "banana")

	inv := m.opts.Inventory
	inv.Groups = append([]config.Group(nil), inv.Groups...)
	inv.Groups[0].Name = "blueberry"
	m.Refresh(inv)

	if got := selectedGroup(m); got != "blueberry" {
		t.Fatalf("selected %q after rename, want blueberry", got)
	}
}

func TestGroupsRefreshFromEmptySelectsFirstRow(t *testing.T) {
	m := groupsModelWith() // nothing selected
	m.Refresh(config.Inventory{Groups: []config.Group{{Name: "mango"}, {Name: "zeta"}, {Name: "alpha"}}})

	if got := selectedGroup(m); got != "alpha" {
		t.Fatalf("selected %q, want alpha (first row, not inventory index 0 or 1)", got)
	}
}
