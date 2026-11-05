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

func TestToastKeyAndLevel(t *testing.T) {
	none := toast{}
	tc := []struct {
		name     string
		h, g, gh toast
		key      string
		level    toastLevel
	}{
		{"no toasts", none, none, none, "", toastInfo},
		{"blank text is no toast", toast{text: "  ", level: toastErr}, none, none, "", toastInfo},
		{"hosts", toast{text: "h", level: toastWarn}, none, none, "h", toastWarn},
		{"groups", none, toast{text: "g", level: toastOK}, none, "|g", toastOK},
		{"group hosts", none, none, toast{text: "x", level: toastErr}, "|x", toastErr},
		{"first is highest", toast{text: "h", level: toastErr}, toast{text: "g", level: toastWarn}, toast{text: "x", level: toastOK}, "h|g|x", toastErr},
		{"middle is highest", toast{text: "h", level: toastOK}, toast{text: "g", level: toastErr}, toast{text: "x", level: toastWarn}, "h|g|x", toastErr},
		{"last is highest", toast{text: "h", level: toastOK}, toast{text: "g", level: toastInfo}, toast{text: "x", level: toastWarn}, "h|g|x", toastWarn},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			m := addedModel(t) // hosts, groups and group hosts are all open
			m.hosts.toast, m.groups.toast, m.gh.toast = c.h, c.g, c.gh
			if key, level := m.collectToastKey(), m.maxToastLevel(); key != c.key || level != c.level {
				t.Fatalf("key=%q level=%v, want %q %v", key, level, c.key, c.level)
			}
		})
	}
}

func TestToastKeySkipsClosedScreens(t *testing.T) {
	m := addedModel(t)
	m.hosts.toast = toast{text: "h", level: toastErr}
	m.groups.toast = toast{text: "g", level: toastWarn}
	m.gh.toast = toast{text: "x", level: toastOK}

	m.hosts = nil
	if key, level := m.collectToastKey(), m.maxToastLevel(); key != "|g|x" || level != toastWarn {
		t.Fatalf("hosts closed: key=%q level=%v", key, level)
	}
	m.groups = nil
	if key, level := m.collectToastKey(), m.maxToastLevel(); key != "|x" || level != toastOK {
		t.Fatalf("groups closed: key=%q level=%v", key, level)
	}
	m.gh = nil
	if key, level := m.collectToastKey(), m.maxToastLevel(); key != "" || level != toastInfo {
		t.Fatalf("all closed: key=%q level=%v", key, level)
	}
}
