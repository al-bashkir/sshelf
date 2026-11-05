package ui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/al-bashkir/sshelf/internal/config"
)

// routerApp is the app in a 100x30 terminal on the known hosts db and web.
// web has a host config; group prod holds web and db and sets user ops,
// group stage is empty and sets user deploy. Saves go to a temp dir;
// connects never use tmux.
func routerApp(t *testing.T) *appModel {
	t.Helper()
	dir := t.TempDir()
	m := newAppModel(Options{
		Config:        config.DefaultConfig(),
		ConfigPath:    filepath.Join(dir, "config.toml"),
		InventoryPath: filepath.Join(dir, "hosts.toml"),
		Inventory: config.Inventory{
			Hosts: []config.Host{{Host: "web", User: "w"}},
			Groups: []config.Group{
				{Name: "prod", User: "ops", Hosts: []string{"web", "db"}},
				{Name: "stage", User: "deploy"},
			},
		},
		Hosts:  []string{"db", "web"},
		NoTmux: true,
	})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

// blockSaves makes every later save fail: both files sit under a regular file.
func blockSaves(t *testing.T, m *appModel) {
	t.Helper()
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	m.opts.ConfigPath = filepath.Join(blocker, "config.toml")
	m.opts.InventoryPath = filepath.Join(blocker, "hosts.toml")
}

func savedInventory(t *testing.T, m *appModel) config.Inventory {
	t.Helper()
	inv, _, err := config.LoadInventory(m.opts.InventoryPath)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func savedGroupHosts(t *testing.T, m *appModel, name string) []string {
	t.Helper()
	for _, g := range savedInventory(t, m).Groups {
		if g.Name == name {
			return g.Hosts
		}
	}
	t.Fatalf("group %q not saved", name)
	return nil
}

func wantScreen(t *testing.T, m *appModel, s screen) {
	t.Helper()
	if m.screen != s {
		t.Fatalf("screen = %v, want %v", m.screen, s)
	}
}

func wantToast(t *testing.T, where string, got toast, text string, level toastLevel) {
	t.Helper()
	if got.text != text || got.level != level {
		t.Fatalf("%s toast = %q (level %v), want %q (level %v)", where, got.text, got.level, text, level)
	}
}

// wantErrToast checks for an error toast; its text comes from the OS or config.
func wantErrToast(t *testing.T, where string, got toast) {
	t.Helper()
	if got.text == "" || got.level != toastErr {
		t.Fatalf("%s toast = %q (level %v), want an error", where, got.text, got.level)
	}
}

func wantSize(t *testing.T, what string, gotW, gotH int, size func(w, h int) (int, int), w, h int) {
	t.Helper()
	if size != nil {
		w, h = size(w, h)
	}
	if gotW != w || gotH != h {
		t.Fatalf("%s size = %dx%d, want %dx%d", what, gotW, gotH, w, h)
	}
}

// rowHosts lists the hosts of list rows, in order.
func rowHosts(items []list.Item) []string {
	var out []string
	for _, it := range items {
		if r, ok := it.(hostRow); ok {
			out = append(out, r.host)
		}
	}
	return out
}

// hasCfgBadge reports whether the row for host shows the host-config badge.
func hasCfgBadge(items []list.Item, host string) bool {
	for _, it := range items {
		if r, ok := it.(hostRow); ok && r.host == host {
			return r.hasCfg
		}
	}
	return false
}

func TestRouterSwitchScreen(t *testing.T) {
	m := routerApp(t)
	m.Update(switchScreenMsg{to: screenGroups})
	wantScreen(t, m, screenGroups)
}

func TestRouterGroupForm(t *testing.T) {
	t.Run("open an existing group", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormMsg{index: 0})
		wantScreen(t, m, screenGroupForm)
		if m.form.index != 0 || m.form.group.Name != "prod" || m.form.parentCrumb != "Groups" {
			t.Fatalf("form index=%d group=%q crumb=%q, want 0 prod Groups", m.form.index, m.form.group.Name, m.form.parentCrumb)
		}
		wantSize(t, "form", m.form.width, m.form.height, groupFormModalSize, 100, 30)
	})
	t.Run("an index out of range opens an empty form", func(t *testing.T) {
		for _, i := range []int{-1, 2} {
			m := routerApp(t)
			m.Update(openGroupFormMsg{index: i})
			if m.form.index != i || m.form.group.Name != "" {
				t.Fatalf("index %d: form index=%d group=%q, want an empty group", i, m.form.index, m.form.group.Name)
			}
		}
	})
	t.Run("prefill opens a new group", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormPrefillMsg{group: config.Group{Name: "copy"}})
		wantScreen(t, m, screenGroupForm)
		if m.form.index != -1 || m.form.group.Name != "copy" || m.form.parentCrumb != "Groups" {
			t.Fatalf("form index=%d group=%q crumb=%q, want -1 copy Groups", m.form.index, m.form.group.Name, m.form.parentCrumb)
		}
		wantSize(t, "form", m.form.width, m.form.height, groupFormModalSize, 100, 30)
	})
	t.Run("a screen opened before any window size gets no size", func(t *testing.T) {
		m := newAppModel(Options{Config: config.DefaultConfig()})
		m.Update(openGroupFormMsg{index: -1})
		wantSize(t, "form", m.form.width, m.form.height, nil, 0, 0)
	})
	t.Run("cancel goes back to Groups", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormMsg{index: 0})
		m.Update(groupFormCancelMsg{})
		wantScreen(t, m, screenGroups)
		if m.form != nil {
			t.Fatal("form still open")
		}
	})
	t.Run("save writes the group and goes back to Groups", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormPrefillMsg{})
		m.Update(groupFormSaveMsg{index: -1, group: config.Group{Name: " qa "}})
		wantScreen(t, m, screenGroups)
		wantToast(t, "groups", m.groups.toast, "saved", toastOK)
		if m.form != nil {
			t.Fatal("form still open")
		}
		if got := savedInventory(t, m).Groups; len(got) != 3 || got[2].Name != "qa" {
			t.Fatalf("saved groups %+v, want a trimmed qa last", got)
		}
	})
	t.Run("save over an existing group replaces it", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormMsg{index: 0})
		m.Update(groupFormSaveMsg{index: 0, group: config.Group{Name: "live"}})
		if got := savedInventory(t, m).Groups; len(got) != 2 || got[0].Name != "live" {
			t.Fatalf("saved groups %+v, want live in place of prod", got)
		}
	})
	t.Run("a failed save keeps the form open", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormMsg{index: 1})
		m.Update(groupFormSaveMsg{index: 1, group: config.Group{Name: "prod"}})
		wantScreen(t, m, screenGroupForm)
		wantToast(t, "form", m.form.toast, "group name already exists", toastErr)
	})
	t.Run("a form opened out of range cannot save", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupFormMsg{index: 2})
		m.Update(groupFormSaveMsg{index: 2, group: config.Group{Name: "qa"}})
		wantScreen(t, m, screenGroupForm)
		wantToast(t, "form", m.form.toast, "invalid group name index", toastErr)
	})
	t.Run("delete", func(t *testing.T) {
		m := routerApp(t)
		m.Update(deleteGroupMsg{index: 0})
		wantToast(t, "groups", m.groups.toast, "deleted", toastOK)
		if got := savedInventory(t, m).Groups; len(got) != 1 || got[0].Name != "stage" {
			t.Fatalf("saved groups %+v, want only stage", got)
		}
	})
	t.Run("a failed delete shows the error", func(t *testing.T) {
		m := routerApp(t)
		m.Update(deleteGroupMsg{index: 2})
		wantToast(t, "groups", m.groups.toast, "invalid group index", toastErr)
	})
}

func TestRouterGroupHosts(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		wantScreen(t, m, screenGroupHosts)
		if m.gh.group.Name != "prod" {
			t.Fatalf("group hosts of %q, want prod", m.gh.group.Name)
		}
		wantSize(t, "group hosts", m.gh.width, m.gh.height, nil, 100, 30)
	})
	t.Run("an index out of range stays put", func(t *testing.T) {
		for _, i := range []int{-1, 2} {
			m := routerApp(t)
			m.Update(openGroupHostsMsg{index: i})
			wantScreen(t, m, screenHosts)
			wantToast(t, "groups", m.groups.toast, "invalid group", toastErr)
			if m.gh != nil {
				t.Fatalf("index %d opened group hosts", i)
			}
		}
	})
	t.Run("remove writes the group and shows what is left", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(removeHostsMsg{groupIndex: 0, hosts: []string{"web"}})
		wantScreen(t, m, screenGroupHosts)
		if got := savedGroupHosts(t, m, "prod"); !slices.Equal(got, []string{"db"}) {
			t.Fatalf("saved prod hosts %v, want [db]", got)
		}
		if got := rowHosts(m.gh.l.Items()); !slices.Equal(got, []string{"db"}) {
			t.Fatalf("group hosts shows %v, want [db]", got)
		}
		wantToast(t, "group hosts", m.gh.toast, "removed 1", toastOK)
		wantSize(t, "group hosts", m.gh.width, m.gh.height, nil, 100, 30)
	})
	t.Run("a failed remove shows the error", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(removeHostsMsg{groupIndex: 2, hosts: []string{"web"}})
		wantErrToast(t, "group hosts", m.gh.toast)
	})
	t.Run("remove from another screen only saves", func(t *testing.T) {
		m := routerApp(t)
		m.Update(removeHostsMsg{groupIndex: 0, hosts: []string{"web"}})
		wantScreen(t, m, screenHosts)
		if m.gh != nil {
			t.Fatal("group hosts opened")
		}
		if got := savedGroupHosts(t, m, "prod"); !slices.Equal(got, []string{"db"}) {
			t.Fatalf("saved prod hosts %v, want [db]", got)
		}
	})
}

func TestRouterHostPicker(t *testing.T) {
	open := func(t *testing.T) *appModel {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 1})
		m.Update(openHostPickerMsg{groupIndex: 1, returnTo: screenGroupHosts})
		return m
	}
	t.Run("open", func(t *testing.T) {
		m := open(t)
		wantScreen(t, m, screenHostPicker)
		if m.picker.parentCrumb != "Groups > stage" {
			t.Fatalf("picker crumb %q, want Groups > stage", m.picker.parentCrumb)
		}
		wantSize(t, "picker", m.picker.width, m.picker.height, pickerModalSize, 100, 30)
	})
	t.Run("cancel goes back", func(t *testing.T) {
		m := open(t)
		m.Update(hostPickerCancelMsg{})
		wantScreen(t, m, screenGroupHosts)
		if m.picker != nil {
			t.Fatal("picker still open")
		}
	})
	t.Run("done adds the hosts and refreshes group hosts", func(t *testing.T) {
		m := open(t)
		m.Update(hostPickerDoneMsg{hosts: []string{"db", "db"}})
		wantScreen(t, m, screenGroupHosts)
		if m.picker != nil {
			t.Fatal("picker still open")
		}
		if got := rowHosts(m.gh.l.Items()); !slices.Equal(got, []string{"db"}) {
			t.Fatalf("group hosts shows %v, want [db]", got)
		}
		wantToast(t, "group hosts", m.gh.toast, "added 1", toastOK)
		wantSize(t, "group hosts", m.gh.width, m.gh.height, nil, 100, 30)
	})
	t.Run("done with no hosts writes nothing", func(t *testing.T) {
		m := open(t)
		m.Update(hostPickerDoneMsg{})
		wantToast(t, "group hosts", m.gh.toast, "added 0", toastOK)
		if _, err := os.Stat(m.opts.InventoryPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inventory file written for no hosts: %v", err)
		}
	})
	t.Run("done from Groups toasts there", func(t *testing.T) {
		m := routerApp(t)
		m.Update(switchScreenMsg{to: screenGroups})
		m.Update(openHostPickerMsg{groupIndex: 0, returnTo: screenGroups})
		m.Update(hostPickerDoneMsg{hosts: []string{"web", "new"}})
		wantScreen(t, m, screenGroups)
		wantToast(t, "groups", m.groups.toast, "added 1", toastOK)
		if m.gh != nil {
			t.Fatal("group hosts opened")
		}
	})
	t.Run("a failed add keeps the picker open", func(t *testing.T) {
		m := open(t)
		m.returnGroupIndex = 2
		m.Update(hostPickerDoneMsg{hosts: []string{"db"}})
		wantScreen(t, m, screenHostPicker)
		wantToast(t, "picker", m.picker.toast, "invalid group index", toastErr)
	})
}

func TestRouterGroupPicker(t *testing.T) {
	open := func(t *testing.T) *appModel {
		m := routerApp(t)
		hs := []string{"db"}
		m.Update(openGroupPickerMsg{hosts: hs})
		hs[0] = "changed" // the picker keeps its own copy
		return m
	}
	t.Run("open", func(t *testing.T) {
		m := open(t)
		wantScreen(t, m, screenGroupPicker)
		if !slices.Equal(m.gp.hosts, []string{"db"}) || m.gp.parentCrumb != "Hosts" {
			t.Fatalf("group picker hosts=%v crumb=%q, want [db] Hosts", m.gp.hosts, m.gp.parentCrumb)
		}
		wantSize(t, "group picker", m.gp.width, m.gp.height, pickerModalSize, 100, 30)
	})
	t.Run("cancel goes back to Hosts", func(t *testing.T) {
		m := open(t)
		m.Update(groupPickerCancelMsg{})
		wantScreen(t, m, screenHosts)
		if m.gp != nil {
			t.Fatal("group picker still open")
		}
	})
	t.Run("done adds the hosts", func(t *testing.T) {
		m := open(t)
		m.Update(groupPickerDoneMsg{groupIndex: 1})
		wantScreen(t, m, screenHosts)
		wantToast(t, "hosts", m.hosts.toast, "added 1", toastOK)
		if m.gp != nil {
			t.Fatal("group picker still open")
		}
		if got := savedGroupHosts(t, m, "stage"); !slices.Equal(got, []string{"db"}) {
			t.Fatalf("saved stage hosts %v, want [db]", got)
		}
	})
	t.Run("a failed add keeps the picker open", func(t *testing.T) {
		m := open(t)
		m.Update(groupPickerDoneMsg{groupIndex: 2})
		wantScreen(t, m, screenGroupPicker)
		wantToast(t, "group picker", m.gp.toast, "invalid group index", toastErr)
	})
}

func TestRouterDefaultsForm(t *testing.T) {
	open := func(t *testing.T) *appModel {
		m := routerApp(t)
		m.Update(switchScreenMsg{to: screenGroups})
		m.Update(openDefaultsFormMsg{returnTo: screenGroups})
		return m
	}
	t.Run("open", func(t *testing.T) {
		m := open(t)
		wantScreen(t, m, screenDefaultsForm)
		wantSize(t, "settings", m.defaultsForm.width, m.defaultsForm.height, nil, 100, 30)
	})
	t.Run("cancel goes back", func(t *testing.T) {
		m := open(t)
		m.Update(defaultsFormCancelMsg{})
		wantScreen(t, m, screenGroups)
		if m.defaultsForm != nil {
			t.Fatal("settings still open")
		}
	})
	t.Run("save writes the config and shows saved for a while", func(t *testing.T) {
		m := open(t)
		m.defaultsForm.form.setValue("user", "edited")
		m.screen = screenHosts // the user left the tab before the save arrived
		d := m.opts.Config.Defaults
		d.ConnectConfirmThreshold = 9
		_, cmd := m.Update(defaultsFormSaveMsg{defaults: d})

		wantScreen(t, m, screenDefaultsForm)
		wantToast(t, "settings", m.defaultsForm.toast, "saved", toastOK)
		if m.opts.Config.Defaults.ConnectConfirmThreshold != 9 || m.defaultsForm.defaults.ConnectConfirmThreshold != 9 {
			t.Fatal("app or form still has the old defaults")
		}
		if m.defaultsForm.form.isDirty() {
			t.Fatal("form still dirty after save")
		}
		if cfg, _, err := config.Load(m.opts.ConfigPath); err != nil || cfg.Defaults.ConnectConfirmThreshold != 9 {
			t.Fatalf("saved threshold %d err=%v, want 9", cfg.Defaults.ConnectConfirmThreshold, err)
		}
		if cmd == nil {
			t.Fatal("no timer to clear the saved toast")
		}

		m.Update(defaultsToastExpireMsg{token: m.defaultsToastToken - 1})
		wantToast(t, "settings after a stale timer", m.defaultsForm.toast, "saved", toastOK)
		m.Update(defaultsToastExpireMsg{token: m.defaultsToastToken})
		wantToast(t, "settings after the timer", m.defaultsForm.toast, "", toastInfo)
	})
	t.Run("the timer keeps a newer toast", func(t *testing.T) {
		m := open(t)
		m.Update(defaultsFormSaveMsg{defaults: m.opts.Config.Defaults})
		m.defaultsForm.toast = toast{text: "other", level: toastWarn}
		m.Update(defaultsToastExpireMsg{token: m.defaultsToastToken})
		wantToast(t, "settings", m.defaultsForm.toast, "other", toastWarn)
	})
	t.Run("turning known_hosts off lists the inventory hosts", func(t *testing.T) {
		m := open(t)
		m.opts.SkippedLines = 2
		d := m.opts.Config.Defaults
		d.LoadKnownHosts = false
		m.Update(defaultsFormSaveMsg{defaults: d})

		want := config.ConfigHosts(m.opts.Inventory)
		if m.opts.KnownHosts != nil || !slices.Equal(m.opts.Hosts, want) || m.opts.SkippedLines != 0 || m.opts.LoadErrors != nil {
			t.Fatalf("known=%v hosts=%v skipped=%d errs=%v, want the inventory hosts %v and nothing else",
				m.opts.KnownHosts, m.opts.Hosts, m.opts.SkippedLines, m.opts.LoadErrors, want)
		}
		if got := shownHosts(m.hosts); !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(want))) {
			t.Fatalf("hosts screen shows %v, want %v", got, want)
		}
		if m.hosts.keymap.Reload.Enabled() {
			t.Fatal("reload key still on without known_hosts")
		}
	})
	t.Run("turning known_hosts on reads the files", func(t *testing.T) {
		m := open(t)
		m.opts.Config.Defaults.LoadKnownHosts = false
		m.hosts.keymap.Reload.SetEnabled(false)
		file := filepath.Join(t.TempDir(), "known_hosts")
		if err := os.WriteFile(file, []byte("alpha ssh-ed25519 AAAAC3NzaC1lZDI1NTE5\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		m.opts.KnownHosts = []string{file}
		d := m.opts.Config.Defaults
		d.LoadKnownHosts = true
		m.Update(defaultsFormSaveMsg{defaults: d})

		if !slices.Equal(m.opts.KnownHosts, []string{file}) || !slices.Equal(m.opts.Hosts, []string{"alpha"}) {
			t.Fatalf("known=%v hosts=%v, want %s read for [alpha]", m.opts.KnownHosts, m.opts.Hosts, file)
		}
		if got := shownHosts(m.hosts); !slices.Equal(got, []string{"alpha"}) {
			t.Fatalf("hosts screen shows %v, want [alpha]", got)
		}
		if !m.hosts.keymap.Reload.Enabled() {
			t.Fatal("reload key off with known_hosts")
		}
	})
	t.Run("turning known_hosts on with no files reads the default file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		file := filepath.Join(home, ".ssh", "known_hosts")
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("beta ssh-ed25519 AAAAC3NzaC1lZDI1NTE5\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		m := open(t)
		m.opts.Config.Defaults.LoadKnownHosts = false
		d := m.opts.Config.Defaults
		d.LoadKnownHosts = true
		m.Update(defaultsFormSaveMsg{defaults: d})

		if !slices.Equal(m.opts.KnownHosts, []string{file}) || !slices.Equal(m.opts.Hosts, []string{"beta"}) {
			t.Fatalf("known=%v hosts=%v, want the default file read for [beta]", m.opts.KnownHosts, m.opts.Hosts)
		}
	})
	t.Run("the new defaults reach every list screen", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(openDefaultsFormMsg{returnTo: screenGroupHosts})
		d := m.opts.Config.Defaults
		d.ConnectConfirmThreshold = 9
		m.Update(defaultsFormSaveMsg{defaults: d})
		for name, got := range map[string]config.Defaults{
			"hosts": m.hosts.opts.Config.Defaults, "groups": m.groups.opts.Config.Defaults, "group hosts": m.gh.opts.Config.Defaults,
		} {
			if got.ConnectConfirmThreshold != 9 {
				t.Fatalf("%s still has threshold %d", name, got.ConnectConfirmThreshold)
			}
		}
	})
	t.Run("a new accent restyles every open input", func(t *testing.T) {
		t.Cleanup(func() { ApplyColorScheme("", "") })
		m := routerApp(t)
		m.hosts.focus, m.hosts.cmdPrompt = focusSearch, true
		m.groups.focus, m.groups.cmdPrompt = focusSearch, true
		m.Update(openGroupHostsMsg{index: 0})
		m.gh.focus, m.gh.cmdPrompt = focusSearch, true
		m.Update(openHostPickerMsg{groupIndex: 0, returnTo: screenGroupHosts})
		m.picker.focus = focusSearch
		m.Update(openGroupPickerMsg{hosts: []string{"db"}})
		m.gp.focus = focusSearch
		m.Update(openCustomHostMsg{returnTo: screenHosts, groupIndex: -1})
		m.Update(openGroupFormMsg{index: 0})
		m.Update(openHostFormMsg{host: "web", returnTo: screenHosts})
		m.Update(openDefaultsFormMsg{returnTo: screenHosts})

		old := cAccent
		d := m.opts.Config.Defaults
		d.AccentColor = "#ff0000"
		m.Update(defaultsFormSaveMsg{defaults: d})
		if cAccent == old {
			t.Fatal("accent did not change")
		}

		accented := func(ti textinput.Model) bool { return ti.PromptStyle.GetForeground() == cAccent }
		anyAccented := func(f formModel) bool {
			return slices.ContainsFunc(f.inputs, func(in formTextInput) bool { return accented(in.input) })
		}
		for name, ok := range map[string]bool{
			"hosts search": accented(m.hosts.search), "hosts command": accented(m.hosts.cmdInput),
			"groups search": accented(m.groups.search), "groups command": accented(m.groups.cmdInput),
			"group hosts search": accented(m.gh.search), "group hosts command": accented(m.gh.cmdInput),
			"picker search": accented(m.picker.search), "group picker search": accented(m.gp.search),
			"custom host": accented(m.customHost.input),
			"group form":  anyAccented(m.form.form), "host form": anyAccented(m.hostForm.form),
			"settings": anyAccented(m.defaultsForm.form),
		} {
			if !ok {
				t.Fatalf("%s does not use the new accent", name)
			}
		}
	})
	t.Run("a failed save keeps the old defaults", func(t *testing.T) {
		m := open(t)
		blockSaves(t, m)
		d := m.opts.Config.Defaults
		d.ConnectConfirmThreshold = 9
		m.Update(defaultsFormSaveMsg{defaults: d})
		wantScreen(t, m, screenDefaultsForm)
		wantErrToast(t, "settings", m.defaultsForm.toast)
		if m.opts.Config.Defaults.ConnectConfirmThreshold == 9 {
			t.Fatal("app took the defaults that failed to save")
		}
	})
}

func TestRouterCustomHost(t *testing.T) {
	t.Run("open names the parent screen", func(t *testing.T) {
		cases := []struct {
			from       screen
			groupIndex int
			want       string
		}{
			{screenHosts, -1, "Hosts"},
			{screenHosts, 0, "Hosts"},
			{screenGroups, 0, "Groups > prod"},
			{screenGroups, 1, "Groups > stage"},
			{screenGroups, -1, "Groups"},
			{screenGroups, 2, "Groups"},
		}
		for _, c := range cases {
			m := routerApp(t)
			m.Update(switchScreenMsg{to: c.from})
			m.Update(openCustomHostMsg{returnTo: c.from, groupIndex: c.groupIndex})
			wantScreen(t, m, screenCustomHost)
			if m.customHost.parentCrumb != c.want || m.customHost.returnTo != c.from || m.customHost.groupIndex != c.groupIndex {
				t.Fatalf("from %v group %d: crumb=%q returnTo=%v group=%d, want %q", c.from, c.groupIndex,
					m.customHost.parentCrumb, m.customHost.returnTo, m.customHost.groupIndex, c.want)
			}
			wantSize(t, "custom host", m.customHost.width, m.customHost.height, customHostModalSize, 100, 30)
		}
	})
	t.Run("open from group hosts names the group", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: 0})
		if m.customHost.parentCrumb != "Groups > prod" {
			t.Fatalf("crumb %q, want Groups > prod", m.customHost.parentCrumb)
		}
	})
	t.Run("cancel goes back", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openCustomHostMsg{returnTo: screenGroups, groupIndex: -1})
		m.Update(customHostCancelMsg{})
		wantScreen(t, m, screenGroups)
		if m.customHost != nil {
			t.Fatal("custom host still open")
		}
	})
	t.Run("cancel with nothing open goes to Hosts", func(t *testing.T) {
		m := routerApp(t)
		m.screen = screenGroups
		m.Update(customHostCancelMsg{})
		wantScreen(t, m, screenHosts)
	})
	t.Run("pick group opens the group picker", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openCustomHostMsg{returnTo: screenGroups, groupIndex: -1})
		hs := []string{"x"}
		m.Update(customHostPickGroupMsg{returnTo: screenGroups, hosts: hs})
		hs[0] = "changed"
		wantScreen(t, m, screenGroupPicker)
		if m.customHost != nil {
			t.Fatal("custom host still open")
		}
		if !slices.Equal(m.gp.hosts, []string{"x"}) || m.gp.parentCrumb != "" {
			t.Fatalf("group picker hosts=%v crumb=%q, want [x] and no crumb", m.gp.hosts, m.gp.parentCrumb)
		}
		wantSize(t, "group picker", m.gp.width, m.gp.height, pickerModalSize, 100, 30)

		m.Update(groupPickerDoneMsg{groupIndex: 1})
		wantScreen(t, m, screenGroups)
		wantToast(t, "groups", m.groups.toast, "added 1", toastOK)
		if got := savedGroupHosts(t, m, "stage"); !slices.Equal(got, []string{"x"}) {
			t.Fatalf("saved stage hosts %v, want [x]", got)
		}
	})
	t.Run("connect execs ssh with the group settings", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openCustomHostMsg{returnTo: screenGroups, groupIndex: 1})
		_, cmd := m.Update(customHostConnectMsg{returnTo: screenGroups, groupIndex: 1, hosts: []string{"x"}})
		wantScreen(t, m, screenGroups)
		if m.customHost != nil {
			t.Fatal("custom host still open")
		}
		argv := strings.Join(m.ExecCmd(), " ")
		if !isQuit(cmd) || !strings.Contains(argv, "x") || !strings.Contains(argv, "deploy") {
			t.Fatalf("quit=%v exec %q, want ssh to x as deploy", isQuit(cmd), argv)
		}
	})
	t.Run("connect with the first group", func(t *testing.T) {
		m := routerApp(t)
		m.Update(customHostConnectMsg{returnTo: screenGroups, groupIndex: 0, hosts: []string{"x"}})
		if argv := strings.Join(m.ExecCmd(), " "); !strings.Contains(argv, "ops") {
			t.Fatalf("exec %q, want ssh as ops", argv)
		}
	})
	t.Run("connect without a group", func(t *testing.T) {
		m := routerApp(t)
		_, cmd := m.Update(customHostConnectMsg{returnTo: screenHosts, groupIndex: -1, hosts: []string{"x"}})
		argv := strings.Join(m.ExecCmd(), " ")
		if !isQuit(cmd) || !strings.Contains(argv, "x") || strings.Contains(argv, "deploy") {
			t.Fatalf("quit=%v exec %q, want ssh to x with no group user", isQuit(cmd), argv)
		}
	})
	t.Run("connect into a missing group", func(t *testing.T) {
		m := routerApp(t)
		// Do not run the returned command: it is the toast timer and sleeps.
		m.Update(customHostConnectMsg{returnTo: screenGroups, groupIndex: 2, hosts: []string{"x"}})
		wantScreen(t, m, screenGroups)
		wantToast(t, "groups", m.groups.toast, "invalid group", toastErr)
		if m.IsQuitting() || len(m.ExecCmd()) != 0 {
			t.Fatal("connected into a missing group")
		}
	})
	t.Run("done adds the hosts and refreshes group hosts", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 1})
		m.Update(openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: 1})
		m.Update(customHostDoneMsg{returnTo: screenGroupHosts, groupIndex: 1, hosts: []string{"new"}})
		wantScreen(t, m, screenGroupHosts)
		if m.customHost != nil {
			t.Fatal("custom host still open")
		}
		if got := rowHosts(m.gh.l.Items()); !slices.Equal(got, []string{"new"}) {
			t.Fatalf("group hosts shows %v, want [new]", got)
		}
		wantToast(t, "group hosts", m.gh.toast, "added 1", toastOK)
		wantSize(t, "group hosts", m.gh.width, m.gh.height, nil, 100, 30)
	})
	t.Run("a failed done keeps the custom host open", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openCustomHostMsg{returnTo: screenHosts, groupIndex: 2})
		m.Update(customHostDoneMsg{returnTo: screenHosts, groupIndex: 2, hosts: []string{"x"}})
		wantScreen(t, m, screenCustomHost)
		wantToast(t, "custom host", m.customHost.toast, "invalid group index", toastErr)
	})
	t.Run("a failed done with the custom host closed", func(t *testing.T) {
		m := routerApp(t)
		m.Update(customHostDoneMsg{returnTo: screenHosts, groupIndex: 9, hosts: []string{"x"}})
		wantScreen(t, m, screenHosts)
	})
}

func TestRouterHostForm(t *testing.T) {
	t.Run("open a host with a config", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openHostFormMsg{host: "web", returnTo: screenHosts})
		wantScreen(t, m, screenHostForm)
		if m.hostForm.index != 0 || m.hostForm.host.User != "w" || m.hostForm.parentCrumb != "Hosts" {
			t.Fatalf("host form index=%d user=%q crumb=%q, want 0 w Hosts", m.hostForm.index, m.hostForm.host.User, m.hostForm.parentCrumb)
		}
		wantSize(t, "host form", m.hostForm.width, m.hostForm.height, hostFormModalSize, 100, 30)
	})
	t.Run("open a host without a config", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openHostFormMsg{host: "db", returnTo: screenHosts})
		if m.hostForm.index != -1 || m.hostForm.host.Host != "db" {
			t.Fatalf("host form index=%d host=%q, want -1 db", m.hostForm.index, m.hostForm.host.Host)
		}
	})
	t.Run("prefill opens a new host", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(openHostFormPrefillMsg{host: config.Host{Host: "copy"}, returnTo: screenGroupHosts})
		wantScreen(t, m, screenHostForm)
		if m.hostForm.index != -1 || m.hostForm.host.Host != "copy" || m.hostForm.parentCrumb != "Groups > prod" {
			t.Fatalf("host form index=%d host=%q crumb=%q, want -1 copy Groups > prod", m.hostForm.index, m.hostForm.host.Host, m.hostForm.parentCrumb)
		}
		wantSize(t, "host form", m.hostForm.width, m.hostForm.height, hostFormModalSize, 100, 30)
		m.Update(hostFormCancelMsg{})
		wantScreen(t, m, screenGroupHosts)
		if m.hostForm != nil {
			t.Fatal("host form still open")
		}
	})
	t.Run("save writes the host and shows the badge everywhere", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openGroupHostsMsg{index: 0})
		m.Update(openHostPickerMsg{groupIndex: 0, returnTo: screenGroupHosts})
		m.Update(openHostFormMsg{host: "db", returnTo: screenGroupHosts})
		m.Update(hostFormSaveMsg{index: -1, host: config.Host{Host: "db", User: "d"}})

		wantScreen(t, m, screenGroupHosts)
		wantToast(t, "group hosts", m.gh.toast, "saved", toastOK)
		if m.hostForm != nil {
			t.Fatal("host form still open")
		}
		if got := savedInventory(t, m).Hosts; len(got) != 2 || got[1].Host != "db" {
			t.Fatalf("saved hosts %+v, want web and db", got)
		}
		for name, items := range map[string][]list.Item{
			"hosts": m.hosts.list.Items(), "group hosts": m.gh.l.Items(), "picker": m.picker.l.Items(),
		} {
			if !hasCfgBadge(items, "db") {
				t.Fatalf("%s: db has no config badge", name)
			}
		}
	})
	t.Run("a failed save keeps the form open", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openHostFormMsg{host: "db", returnTo: screenHosts})
		m.Update(hostFormSaveMsg{index: -1, host: config.Host{Host: "web"}})
		wantScreen(t, m, screenHostForm)
		wantToast(t, "host form", m.hostForm.toast, "host already exists", toastErr)
	})
	t.Run("a failed save with the form closed", func(t *testing.T) {
		m := routerApp(t)
		m.Update(hostFormSaveMsg{index: -1, host: config.Host{Host: "web"}})
		wantScreen(t, m, screenHosts)
	})
}

func TestRouterToggleHidden(t *testing.T) {
	t.Run("hide", func(t *testing.T) {
		m := routerApp(t)
		m.Update(toggleHiddenHostMsg{host: "db", hide: true})
		wantToast(t, "hosts", m.hosts.toast, "saved", toastOK)
		if got := savedInventory(t, m).HiddenHosts; !slices.Equal(got, []string{"db"}) {
			t.Fatalf("saved hidden hosts %v, want [db]", got)
		}
		if got := shownHosts(m.hosts); !slices.Equal(got, []string{"web"}) {
			t.Fatalf("hosts screen shows %v, want [web]", got)
		}
	})
	t.Run("hide a host with a config marks the config", func(t *testing.T) {
		m := routerApp(t)
		m.Update(toggleHiddenHostMsg{host: "web", hide: true})
		inv := savedInventory(t, m)
		if len(inv.HiddenHosts) != 0 || len(inv.Hosts) != 1 || !inv.Hosts[0].Hidden {
			t.Fatalf("saved hidden hosts %v, hosts %+v, want web hidden in its config", inv.HiddenHosts, inv.Hosts)
		}
	})
	t.Run("hide an already hidden host keeps one entry", func(t *testing.T) {
		m := routerApp(t)
		m.opts.Inventory.HiddenHosts = []string{" db "}
		m.Update(toggleHiddenHostMsg{host: "db", hide: true})
		if got := savedInventory(t, m).HiddenHosts; !slices.Equal(got, []string{" db "}) {
			t.Fatalf("saved hidden hosts %q, want the one old entry", got)
		}
	})
	t.Run("hide a [host]:port entry flags its host's config", func(t *testing.T) {
		m := routerApp(t)
		m.Update(toggleHiddenHostMsg{host: "[web]:22", hide: true})
		inv := savedInventory(t, m)
		if len(inv.HiddenHosts) != 0 || len(inv.Hosts) != 1 || !inv.Hosts[0].Hidden {
			t.Fatalf("saved hidden hosts %v, hosts %+v, want web hidden in its config", inv.HiddenHosts, inv.Hosts)
		}
	})
	t.Run("show takes every match out of the list", func(t *testing.T) {
		m := routerApp(t)
		m.opts.Inventory.HiddenHosts = []string{"db", "x", " db "}
		m.Update(toggleHiddenHostMsg{host: "db", hide: false})
		wantToast(t, "hosts", m.hosts.toast, "saved", toastOK)
		if got := savedInventory(t, m).HiddenHosts; !slices.Equal(got, []string{"x"}) {
			t.Fatalf("saved hidden hosts %q, want [x]", got)
		}
		if got := shownHosts(m.hosts); !slices.Equal(got, []string{"db", "web"}) {
			t.Fatalf("hosts screen shows %v, want [db web]", got)
		}
	})
	t.Run("show clears the flag in a host config", func(t *testing.T) {
		m := routerApp(t)
		m.opts.Inventory.Hosts[0].Hidden = true
		m.Update(toggleHiddenHostMsg{host: "web", hide: false})
		if got := savedInventory(t, m).Hosts; len(got) != 1 || got[0].Hidden || got[0].User != "w" {
			t.Fatalf("saved hosts %+v, want web shown with its config kept", got)
		}
	})
	t.Run("a failed save shows the error", func(t *testing.T) {
		m := routerApp(t)
		blockSaves(t, m)
		m.Update(toggleHiddenHostMsg{host: "db", hide: true})
		wantErrToast(t, "hosts", m.hosts.toast)
	})
}

func TestRouterSizesEveryOpenScreen(t *testing.T) {
	m := routerApp(t)
	m.Update(openGroupFormMsg{index: 0})
	m.Update(openGroupHostsMsg{index: 0})
	m.Update(openHostPickerMsg{groupIndex: 0, returnTo: screenGroupHosts})
	m.Update(openGroupPickerMsg{hosts: []string{"db"}})
	m.Update(openDefaultsFormMsg{returnTo: screenHosts})
	m.Update(openCustomHostMsg{returnTo: screenHosts, groupIndex: -1})
	m.Update(openHostFormMsg{host: "web", returnTo: screenHosts})

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	wantSize(t, "app", m.width, m.height, nil, 120, 40)
	wantSize(t, "hosts", m.hosts.width, m.hosts.height, nil, 120, 40)
	wantSize(t, "groups", m.groups.width, m.groups.height, nil, 120, 40)
	wantSize(t, "group form", m.form.width, m.form.height, groupFormModalSize, 120, 40)
	wantSize(t, "group hosts", m.gh.width, m.gh.height, nil, 120, 40)
	wantSize(t, "picker", m.picker.width, m.picker.height, pickerModalSize, 120, 40)
	wantSize(t, "group picker", m.gp.width, m.gp.height, pickerModalSize, 120, 40)
	wantSize(t, "settings", m.defaultsForm.width, m.defaultsForm.height, nil, 120, 40)
	wantSize(t, "custom host", m.customHost.width, m.customHost.height, customHostModalSize, 120, 40)
	wantSize(t, "host form", m.hostForm.width, m.hostForm.height, hostFormModalSize, 120, 40)
}

func TestRouterForwardsOtherMessagesToTheScreen(t *testing.T) {
	t.Run("hosts: connect in this pane execs ssh", func(t *testing.T) {
		m := routerApp(t)
		_, cmd := m.Update(hk("O"))
		if !isQuit(cmd) || !slices.Contains(m.ExecCmd(), "db") || m.IsQuitting() {
			t.Fatalf("quit=%v exec=%v quitting=%v, want exec ssh db", isQuit(cmd), m.ExecCmd(), m.IsQuitting())
		}
	})
	t.Run("hosts: q quits", func(t *testing.T) {
		m := routerApp(t)
		_, cmd := m.Update(hk("q"))
		if !isQuit(cmd) || !m.IsQuitting() || len(m.ExecCmd()) != 0 {
			t.Fatalf("quit=%v quitting=%v exec=%v, want a plain quit", isQuit(cmd), m.IsQuitting(), m.ExecCmd())
		}
	})
	t.Run("groups: q quits", func(t *testing.T) {
		m := routerApp(t)
		m.Update(switchScreenMsg{to: screenGroups})
		_, cmd := m.Update(hk("q"))
		if !isQuit(cmd) || !m.IsQuitting() {
			t.Fatalf("quit=%v quitting=%v, want a quit", isQuit(cmd), m.IsQuitting())
		}
	})
	t.Run("groups: connect in this pane execs ssh", func(t *testing.T) {
		m := routerApp(t)
		m.Update(switchScreenMsg{to: screenGroups})
		m.Update(openGroupHostsMsg{index: 0})
		_, cmd := m.Update(hk("O"))
		if !isQuit(cmd) || len(m.ExecCmd()) == 0 || m.IsQuitting() {
			t.Fatalf("quit=%v exec=%v quitting=%v, want exec ssh", isQuit(cmd), m.ExecCmd(), m.IsQuitting())
		}
	})
	t.Run("group hosts: a popup toast quits", func(t *testing.T) {
		m := routerApp(t)
		m.opts.Popup = true
		m.Update(openGroupHostsMsg{index: 0})
		m.gh.opts.Popup = true
		// The new toast batches the quit with its timer, so check the flag.
		m.Update(toastMsg{text: "opened", level: toastOK})
		if !m.IsQuitting() {
			t.Fatal("app not quitting after a popup toast")
		}
	})
	t.Run("modals: esc cancels", func(t *testing.T) {
		cases := []struct {
			open tea.Msg
			want tea.Msg
		}{
			{openGroupFormMsg{index: 0}, groupFormCancelMsg{}},
			{openHostPickerMsg{groupIndex: 0, returnTo: screenHosts}, hostPickerCancelMsg{}},
			{openGroupPickerMsg{hosts: []string{"db"}}, groupPickerCancelMsg{}},
			{openCustomHostMsg{returnTo: screenHosts, groupIndex: -1}, customHostCancelMsg{}},
			{openHostFormMsg{host: "web", returnTo: screenHosts}, hostFormCancelMsg{}},
		}
		for _, c := range cases {
			m := routerApp(t)
			m.Update(c.open)
			_, cmd := m.Update(hk("esc"))
			if got := msgOf(cmd); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("after %T, esc gave %#v, want %#v", c.open, got, c.want)
			}
		}
	})
	t.Run("settings: alt+1 goes to Hosts", func(t *testing.T) {
		m := routerApp(t)
		m.Update(openDefaultsFormMsg{returnTo: screenHosts})
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1"), Alt: true})
		if got := msgOf(cmd); !reflect.DeepEqual(got, switchScreenMsg{to: screenHosts}) {
			t.Fatalf("alt+1 gave %#v, want a switch to Hosts", got)
		}
	})
	t.Run("an unknown screen drops the message", func(t *testing.T) {
		m := routerApp(t)
		m.screen = screen(99)
		if _, cmd := m.Update(hk("q")); cmd != nil || m.IsQuitting() {
			t.Fatal("an unknown screen handled a key")
		}
	})
}

func TestRouterView(t *testing.T) {
	// No colorscheme is active in tests, so the theme background adds nothing.
	t.Run("a tab screen fills the window itself", func(t *testing.T) {
		cases := []struct {
			open tea.Msg
			view func(m *appModel) string
		}{
			{nil, func(m *appModel) string { return m.hosts.View() }},
			{switchScreenMsg{to: screenGroups}, func(m *appModel) string { return m.groups.View() }},
			{openGroupHostsMsg{index: 0}, func(m *appModel) string { return m.gh.View() }},
			{openDefaultsFormMsg{returnTo: screenHosts}, func(m *appModel) string { return m.defaultsForm.View() }},
		}
		for _, c := range cases {
			m := routerApp(t)
			if c.open != nil {
				m.Update(c.open)
			}
			if got, want := m.View(), c.view(m); got != want {
				t.Fatalf("screen %v: view is not the screen's own view", m.screen)
			}
		}
	})
	t.Run("a modal is centered in the window", func(t *testing.T) {
		cases := []struct {
			open tea.Msg
			view func(m *appModel) string
		}{
			{openGroupFormMsg{index: 0}, func(m *appModel) string { return m.form.View() }},
			{openHostPickerMsg{groupIndex: 0, returnTo: screenHosts}, func(m *appModel) string { return m.picker.View() }},
			{openGroupPickerMsg{hosts: []string{"db"}}, func(m *appModel) string { return m.gp.View() }},
			{openCustomHostMsg{returnTo: screenHosts, groupIndex: -1}, func(m *appModel) string { return m.customHost.View() }},
			{openHostFormMsg{host: "web", returnTo: screenHosts}, func(m *appModel) string { return m.hostForm.View() }},
		}
		for _, c := range cases {
			m := routerApp(t)
			m.Update(c.open)
			got := m.View()
			if got != placeCentered(100, 30, c.view(m)) {
				t.Fatalf("screen %v: view is not the modal centered in 100x30", m.screen)
			}
			if lines := strings.Split(got, "\n"); len(lines) != 30 || strings.TrimSpace(lines[0]) != "" {
				t.Fatalf("screen %v: %d lines, first %q; want 30 lines and a blank margin on top", m.screen, len(lines), lines[0])
			}
		}
	})
	t.Run("an unknown screen shows Hosts", func(t *testing.T) {
		m := routerApp(t)
		m.screen = screen(99)
		if m.View() != m.hosts.View() {
			t.Fatal("unknown screen does not show the hosts view")
		}
	})
}

func TestRouterToastTimer(t *testing.T) {
	m := routerApp(t)
	m.Update(openGroupHostsMsg{index: 1})
	m.Update(switchScreenMsg{to: screenGroups})
	if m.toastToken != 0 {
		t.Fatalf("token %d before any toast, want 0", m.toastToken)
	}

	if _, cmd := m.Update(deleteGroupMsg{index: 0}); m.toastToken != 1 || cmd == nil {
		t.Fatalf("new toast: token %d, cmd nil=%v; want a timer with token 1", m.toastToken, cmd == nil)
	}
	if _, cmd := m.Update(switchScreenMsg{to: screenGroups}); m.toastToken != 1 || cmd != nil {
		t.Fatal("a message that shows no new toast started a timer")
	}
	if _, cmd := m.Update(deleteGroupMsg{index: 0}); m.toastToken != 1 || cmd != nil {
		t.Fatal("the same toast text started a second timer")
	}
	m.Update(deleteGroupMsg{index: 0}) // no groups left: an error toast
	if m.toastToken != 2 {
		t.Fatalf("changed toast: token %d, want 2", m.toastToken)
	}

	m.hosts.toast = toast{text: "h", level: toastOK}
	m.gh.toast = toast{text: "g", level: toastOK}
	m.Update(toastDismissMsg{token: 1})
	wantToast(t, "groups after a stale timer", m.groups.toast, "invalid group index", toastErr)

	if _, cmd := m.Update(toastDismissMsg{token: 2}); cmd != nil || m.toastToken != 2 {
		t.Fatal("clearing the toasts started a timer")
	}
	for name, got := range map[string]toast{"hosts": m.hosts.toast, "groups": m.groups.toast, "group hosts": m.gh.toast} {
		if !got.empty() {
			t.Fatalf("%s toast %q not cleared", name, got.text)
		}
	}
}

func TestRouterSpinnerTick(t *testing.T) {
	active, index, minEnd := spinnerActive, spinnerIndex, spinnerMinEnd
	t.Cleanup(func() { spinnerActive, spinnerIndex, spinnerMinEnd = active, index, minEnd })
	past, future := time.Now().Add(-time.Second), time.Now().Add(time.Hour)

	cases := []struct {
		name       string
		active     bool
		minEnd     time.Time
		reloading  bool
		wantTick   bool
		wantActive bool
	}{
		{"off: nothing", false, past, false, false, false},
		{"in its minimum time: keeps ticking", true, future, false, true, true},
		{"minimum time over, reload done: stops", true, past, false, false, false},
		{"minimum time over, still reloading: keeps ticking", true, past, true, true, true},
		{"no minimum time: keeps ticking", true, time.Time{}, false, true, true},
	}
	for _, c := range cases {
		m := routerApp(t)
		spinnerActive, spinnerIndex, spinnerMinEnd = c.active, 0, c.minEnd
		m.hosts.reloading = c.reloading
		_, cmd := m.Update(spinnerTickMsg{})
		if (cmd != nil) != c.wantTick || spinnerActive != c.wantActive {
			t.Fatalf("%s: tick=%v active=%v, want tick=%v active=%v", c.name, cmd != nil, spinnerActive, c.wantTick, c.wantActive)
		}
		if wantIndex := map[bool]int{true: 1, false: 0}[c.active]; spinnerIndex != wantIndex {
			t.Fatalf("%s: frame %d, want %d", c.name, spinnerIndex, wantIndex)
		}
	}
}
