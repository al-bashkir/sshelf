package ui

import (
	"path/filepath"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func addedModel(t *testing.T) *appModel {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Defaults.LoadKnownHosts = false
	m := newAppModel(Options{
		Config:        cfg,
		InventoryPath: filepath.Join(t.TempDir(), "hosts.toml"),
		Inventory:     config.Inventory{Groups: []config.Group{{Name: "prod", Hosts: []string{"a"}}}},
		Hosts:         []string{"a", "b"},
	})
	m.Update(openGroupHostsMsg{index: 0})
	return m
}

func TestHostPickerDoneToastGoesToGroupHostsAndCountsNewOnly(t *testing.T) {
	m := addedModel(t)
	m.returnTo, m.returnGroupIndex = screenGroupHosts, 0

	m.Update(hostPickerDoneMsg{hosts: []string{"a", "b"}}) // "a" is already in the group

	if m.screen != screenGroupHosts {
		t.Fatalf("screen=%v, want group hosts", m.screen)
	}
	if got := m.gh.toast.text; got != "added 1" {
		t.Fatalf("group hosts toast=%q, want %q", got, "added 1")
	}
}

func TestCustomHostDoneToastCountsNewOnly(t *testing.T) {
	m := addedModel(t)

	m.Update(customHostDoneMsg{returnTo: screenHosts, groupIndex: 0, hosts: []string{"a", "b"}})

	if got := m.hosts.toast.text; got != "added 1" {
		t.Fatalf("hosts toast=%q, want %q", got, "added 1")
	}
}
