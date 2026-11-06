package ui

import (
	"reflect"
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/al-bashkir/sshelf/internal/config"
)

// subgroups opens prod (alpha, beta, gamma) with the subgroups web (w1,
// user www) and db (d1, d2) in it: groups 2 and 3.
func subgroups() *groupHostsModel {
	return groupHostsScreen(0, func(o *Options) {
		o.Inventory.Groups = append(o.Inventory.Groups,
			config.Group{Name: "web", Parent: "prod", User: "www", Hosts: []string{"w1"}},
			config.Group{Name: "db", Parent: "prod", Hosts: []string{"d1", "d2"}})
	})
}

// ghRows lists the rows: a group as "name/", a host by name.
func ghRows(m *groupHostsModel) []string {
	var out []string
	for _, it := range m.list.Items() {
		switch r := it.(type) {
		case groupRow:
			out = append(out, r.name+"/")
		case hostRow:
			out = append(out, r.host)
		}
	}
	return out
}

func TestGroupHostsListsSubgroupsFirst(t *testing.T) {
	m := subgroups()
	if got, want := ghRows(m), []string{"db/", "web/", "alpha", "beta", "gamma"}; !slices.Equal(got, want) {
		t.Fatalf("rows %v, want %v", got, want)
	}
	if r, ok := m.list.SelectedItem().(groupRow); !ok || r.name != "db" {
		t.Fatalf("cursor on %v, want db", m.list.SelectedItem())
	}
	// A subgroup row looks like a row of the Groups tab: name and host count.
	screenShows(t, m, []string{"│▸ db  2 ", "│  web  1 ", "│  ◻ alpha"})

	t.Run("a search filters groups and hosts, groups first", func(t *testing.T) {
		m := subgroups()
		on(m, "/", "e")
		if got := ghRows(m); !slices.Equal(got, []string{"web/", "beta"}) {
			t.Fatalf("rows %v, want [web/ beta]", got)
		}
	})
	t.Run("a search keeps the cursor on its host", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "j", "j", "/", "e")
		if m.currentHost() != "beta" {
			t.Fatalf("cursor on %v, want beta", m.list.SelectedItem())
		}
	})
	t.Run("a search keeps the cursor on its group", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "/", "e")
		if r, ok := m.list.SelectedItem().(groupRow); !ok || r.name != "web" {
			t.Fatalf("cursor on %v, want web", m.list.SelectedItem())
		}
	})
	t.Run("clearing a search keeps the cursor on its group", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "/", "e", "esc") // "e" shows web alone, then all rows again
		if r, ok := m.list.SelectedItem().(groupRow); !ok || r.name != "web" {
			t.Fatalf("cursor on %v, want web", m.list.SelectedItem())
		}
	})
	t.Run("clearing a search keeps the cursor on its host", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "j", "j", "j", "/", "a", "esc")
		if m.currentHost() != "gamma" {
			t.Fatalf("cursor on %v, want gamma", m.list.SelectedItem())
		}
	})
	t.Run("a host the search hides leaves the cursor on a row that shows", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "j", "j", "j", "/", "w")
		if r, ok := m.list.SelectedItem().(groupRow); !ok || r.name != "web" {
			t.Fatalf("cursor on %v, want web, the only row", m.list.SelectedItem())
		}
	})
	t.Run("only hosts can be selected", func(t *testing.T) {
		m := subgroups()
		on(m, "space")
		if len(m.selected) != 0 {
			t.Fatalf("space on a group selected %v", m.selected)
		}
		on(m, "ctrl+a")
		if got := m.selectedHosts(); !slices.Equal(got, []string{"alpha", "beta", "gamma"}) {
			t.Fatalf("ctrl+a selected %v, want the hosts", got)
		}
	})
}

func TestGroupHostsSubgroupKeys(t *testing.T) {
	inv := subgroups().opts.Inventory
	dbCopy := inv.Groups[3]
	dbCopy.Name = suggestCopyGroupName(inv, "db")
	newIn := openGroupFormPrefillMsg{group: config.Group{Parent: "prod"}, returnTo: screenGroupHosts}
	for _, c := range []struct {
		keys []string
		want tea.Msg
	}{
		{[]string{"enter"}, openGroupHostsMsg{index: 3}},
		{[]string{"e"}, openGroupFormMsg{index: 3, returnTo: screenGroupHosts}},
		{[]string{"y"}, openGroupFormPrefillMsg{group: dbCopy, returnTo: screenGroupHosts}},
		{[]string{"n"}, newIn},
		{[]string{"j", "j", "n"}, newIn}, // on a host row too
		{[]string{"d", "y"}, deleteGroupMsg{index: 3}},
	} {
		m := subgroups()
		if got := msgOf(on(m, c.keys...)); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%v gave %#v, want %#v", c.keys, got, c.want)
		}
	}
	t.Run("d asks before it deletes", func(t *testing.T) {
		m := subgroups()
		on(m, "d")
		screenShows(t, m, []string{"Delete group?", `Delete "db" (2)?`})
		if cmd := on(m, "n"); cmd != nil || m.confirmDelete {
			t.Fatal("n did not cancel the delete")
		}
	})
}

func TestGroupHostsConnectsASubgroupRow(t *testing.T) {
	t.Run("C on a group row connects that group", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "C")
		if want := []string{"ssh", "--", "www@w1"}; !slices.Equal(m.execCmd, want) {
			t.Fatalf("exec %q, want %q (toast %q)", m.execCmd, want, m.toast.text)
		}
	})
	t.Run("C on a host row connects the whole group", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "j", "C")
		if !m.confirmConnect || m.confirmConnectCount != 6 {
			t.Fatalf("confirm=%v count=%d, want a confirm for prod's 6 hosts", m.confirmConnect, m.confirmConnectCount)
		}
	})
	t.Run("o on a group row opens that group in panes", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "o")
		if len(m.execCmd) != 0 || m.toast.text != "requires an active tmux session" {
			t.Fatalf("exec %q toast %q, want the one-window tmux warning", m.execCmd, m.toast.text)
		}
	})
	t.Run("Ctrl+o on a group row runs on that group", func(t *testing.T) {
		m := subgroups()
		on(m, "j", "ctrl+o")
		if m.cmdPromptCrumb != "Groups > prod > web" {
			t.Fatalf("crumb %q, want the web trail", m.cmdPromptCrumb)
		}
		typeText(m, "ls")
		on(m, "enter")
		if !slices.Contains(m.execCmd, "www@w1") {
			t.Fatalf("exec %q, want www@w1", m.execCmd)
		}
	})
}
