package ui

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"
)

func shownHosts(m *hostsModel) []string {
	var out []string
	for _, it := range m.list.Items() {
		out = append(out, it.FilterValue())
	}
	return out
}

func TestSaveGroupFailureLeavesInventoryUnchanged(t *testing.T) {
	// A regular file where the config dir should be makes every save fail.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	inv := config.Inventory{Groups: []config.Group{{Name: "prod", User: "old"}}}
	m := newAppModel(Options{InventoryPath: filepath.Join(blocker, "hosts.toml"), Inventory: inv})

	if err := m.saveGroup(0, config.Group{Name: "prod", User: "new"}); err == nil {
		t.Fatal("save into a path under a regular file succeeded")
	}
	if got := m.opts.Inventory.Groups[0].User; got != "old" {
		t.Fatalf("after failed save, group user=%q, want old", got)
	}
}

func TestAddHostsToGroupShowsNewHostWithoutKnownHosts(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Defaults.LoadKnownHosts = false
	opts := Options{
		Config:        cfg,
		InventoryPath: filepath.Join(t.TempDir(), "hosts.toml"),
		Inventory:     config.Inventory{Groups: []config.Group{{Name: "prod", Hosts: []string{"old"}}}},
		Hosts:         []string{"old"},
	}
	m := newAppModel(opts)

	if _, err := m.addHostsToGroup(0, []string{"newbox"}); err != nil {
		t.Fatal(err)
	}
	if got := shownHosts(m.hosts); !slices.Equal(got, []string{"newbox", "old"}) {
		t.Fatalf("hosts screen shows %v, want [newbox old]", got)
	}
	if !slices.Contains(m.opts.Hosts, "newbox") {
		t.Fatalf("app hosts %v miss newbox; a new host picker would not list it", m.opts.Hosts)
	}
}

func TestStaleInstanceCannotOverwriteAnotherInstancesSave(t *testing.T) {
	opts := Options{Config: config.DefaultConfig(), InventoryPath: filepath.Join(t.TempDir(), "hosts.toml")}
	a, b := newAppModel(opts), newAppModel(opts)

	if err := a.saveToggleHidden("web", true); err != nil {
		t.Fatal(err)
	}
	if err := a.saveToggleHidden("db", true); err != nil {
		t.Fatalf("second save by the same instance: %v", err)
	}
	if err := b.saveToggleHidden("mail", true); !errors.Is(err, config.ErrChangedOnDisk) {
		t.Fatalf("stale instance save: err=%v, want ErrChangedOnDisk", err)
	}
	inv, _, err := config.LoadInventory(opts.InventoryPath)
	if err != nil || !slices.Equal(inv.HiddenHosts, []string{"web", "db"}) {
		t.Fatalf("file hidden_hosts=%v err=%v, want [web db]", inv.HiddenHosts, err)
	}
}

func TestReloadFinishingOnAnotherScreenStillUpdatesHosts(t *testing.T) {
	m := newAppModel(Options{Config: config.DefaultConfig(), Hosts: []string{"old"}})
	m.hosts.reloading = true // user pressed r on Hosts ...
	m.screen = screenGroups  // ... then switched to Groups before the read finished

	m.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: []string{"a", "b"}}})

	if m.hosts.reloading {
		t.Fatal("hosts screen still reloading; spinner would tick forever")
	}
	if got := shownHosts(m.hosts); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("hosts screen shows %v, want [a b]", got)
	}
	if !slices.Equal(m.opts.Hosts, []string{"a", "b"}) {
		t.Fatalf("app hosts %v, want [a b]; a new host picker would list stale hosts", m.opts.Hosts)
	}
}

func TestCustomHostConnectAboveThresholdAsksFirst(t *testing.T) {
	m := newAppModel(Options{Config: config.DefaultConfig()}) // threshold 5
	many := []string{"h1", "h2", "h3", "h4", "h5", "h6", "h7"}

	m.Update(customHostConnectMsg{returnTo: screenHosts, groupIndex: -1, hosts: many})

	if !m.hosts.confirmConnect || m.hosts.confirmConnectCount != len(many) {
		t.Fatalf("confirm=%v count=%d, want a confirm for %d hosts", m.hosts.confirmConnect, m.hosts.confirmConnectCount, len(many))
	}
	if m.screen != screenHosts {
		t.Fatalf("screen=%v, want the Hosts screen that shows the confirm", m.screen)
	}
}

func TestSavePathsWriteTheFile(t *testing.T) {
	newApp := func(t *testing.T) *appModel {
		return newAppModel(Options{
			Config:        config.DefaultConfig(),
			InventoryPath: filepath.Join(t.TempDir(), "hosts.toml"),
			Inventory: config.Inventory{
				Hosts:  []config.Host{{Host: "web", User: "w"}},
				Groups: []config.Group{{Name: "a", Hosts: []string{"web", "db"}}, {Name: "b"}},
			},
		})
	}
	saved := func(t *testing.T, m *appModel) config.Inventory {
		inv, _, err := config.LoadInventory(m.opts.InventoryPath)
		if err != nil {
			t.Fatal(err)
		}
		return inv
	}

	t.Run("add host config, reject duplicate", func(t *testing.T) {
		m := newApp(t)
		if err := m.saveHostConfig(-1, config.Host{Host: " db ", User: "d"}); err != nil {
			t.Fatal(err)
		}
		if err := m.saveHostConfig(-1, config.Host{Host: "web"}); err == nil {
			t.Fatal("duplicate host config accepted")
		}
		if got := saved(t, m).Hosts; len(got) != 2 || got[1].Host != "db" || got[1].User != "d" {
			t.Fatalf("saved hosts %+v, want web and trimmed db", got)
		}
	})
	t.Run("delete group", func(t *testing.T) {
		m := newApp(t)
		if err := m.deleteGroup(0); err != nil {
			t.Fatal(err)
		}
		if got := saved(t, m).Groups; len(got) != 1 || got[0].Name != "b" {
			t.Fatalf("saved groups %+v, want only b", got)
		}
	})
	t.Run("remove hosts from group", func(t *testing.T) {
		m := newApp(t)
		if err := m.removeHostsFromGroup(0, []string{"web"}); err != nil {
			t.Fatal(err)
		}
		if got := saved(t, m).Groups[0].Hosts; !slices.Equal(got, []string{"db"}) {
			t.Fatalf("saved group hosts %v, want [db]", got)
		}
	})
}
