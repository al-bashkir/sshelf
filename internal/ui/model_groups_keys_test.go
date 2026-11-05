package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/al-bashkir/sshelf/internal/config"
)

// groupsScreen opens the groups tab in a 100x30 terminal on the groups web
// (w1), db (d1 d2, user dba) and empty. Rows sort by name: db, empty, web,
// and the cursor starts on db. Connects never use tmux.
func groupsScreen(edit func(*Options)) *groupsModel {
	o := Options{Config: config.DefaultConfig(), NoTmux: true, Inventory: config.Inventory{Groups: []config.Group{
		{Name: "web", Hosts: []string{"w1"}},
		{Name: "db", User: "dba", Hosts: []string{"d1", "d2"}},
		{Name: "empty"},
	}}}
	o.Config.Defaults.ConfirmQuit = false
	if edit != nil {
		edit(&o)
	}
	m := newGroupsModel(o)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

func noGroups(o *Options) { o.Inventory = config.Inventory{} }

func shownGroups(m *groupsModel) []string {
	var out []string
	for _, it := range m.list.Items() {
		out = append(out, it.(groupRow).name)
	}
	return out
}

func currentGroup(m *groupsModel) string {
	if row, ok := m.list.SelectedItem().(groupRow); ok {
		return row.name
	}
	return ""
}

func TestGroupsKeysOpenOtherScreens(t *testing.T) {
	db := config.Group{Name: "db", User: "dba", Hosts: []string{"d1", "d2"}}
	dbCopy := db
	dbCopy.Name = suggestCopyGroupName(groupsScreen(nil).opts.Inventory, "db")
	cases := []struct {
		key  tea.KeyMsg
		want tea.Msg
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1"), Alt: true}, switchScreenMsg{to: screenHosts}},
		{hk("alt+3"), openDefaultsFormMsg{returnTo: screenGroups}},
		{hk("n"), openGroupFormMsg{index: -1}},
		{hk("N"), openGroupFormMsg{index: -1}},
		{hk("e"), openGroupFormMsg{index: 1}},
		{hk("a"), openHostPickerMsg{groupIndex: 1, returnTo: screenGroups}},
		{hk("c"), openCustomHostMsg{returnTo: screenGroups, groupIndex: 1}},
		{hk("enter"), openGroupHostsMsg{index: 1}},
		{hk("y"), openGroupFormPrefillMsg{group: dbCopy}},
	}
	for _, c := range cases {
		m := groupsScreen(nil)
		_, cmd := m.Update(c.key)
		if got := msgOf(cmd); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%q gave %#v, want %#v", c.key.String(), got, c.want)
		}
	}
}

func TestGroupsKeysWithoutGroup(t *testing.T) {
	for _, k := range []string{"e", "a", "c", "enter", "d", "ctrl+o"} {
		m := groupsScreen(noGroups)
		if cmd := on(m, k); cmd != nil || m.confirmDelete || m.cmdPrompt {
			t.Fatalf("%q with no group: cmd=%v delete=%v prompt=%v, want nothing", k, cmd != nil, m.confirmDelete, m.cmdPrompt)
		}
	}
	for _, k := range []string{"y", "C", "o"} {
		m := groupsScreen(noGroups)
		if cmd := on(m, k); cmd != nil || m.toast.text != "no group selected" {
			t.Fatalf("%q with no group: toast %q, want no group selected", k, m.toast.text)
		}
	}
}

func TestGroupsListKeysTypeInSearch(t *testing.T) {
	for _, k := range []string{"n", "e", "a", "c", "d", "C", "o", "y", "q"} {
		m := groupsScreen(nil)
		on(m, "/", k)
		if m.search.Value() != k || m.confirmDelete || m.quitting || len(m.execCmd) != 0 {
			t.Fatalf("%q in search: query %q, delete=%v quitting=%v; want it typed", k, m.search.Value(), m.confirmDelete, m.quitting)
		}
	}
	m := groupsScreen(nil)
	if on(m, "/", "ctrl+o"); m.cmdPrompt {
		t.Fatal("ctrl+o in search opened the command prompt")
	}
}

func TestGroupsDelete(t *testing.T) {
	ask := func() *groupsModel {
		m := groupsScreen(nil)
		if cmd := on(m, "d"); cmd != nil || !m.confirmDelete {
			t.Fatal("d did not ask")
		}
		wantToast(t, "groups", m.toast, "delete? (y/n)", toastWarn)
		return m
	}
	for _, k := range []string{"y", "Y", "enter"} {
		m := ask()
		if got := msgOf(on(m, k)); !reflect.DeepEqual(got, deleteGroupMsg{index: 1}) || m.confirmDelete {
			t.Fatalf("%q gave %#v, want delete of db", k, got)
		}
	}
	for _, k := range []string{"n", "N", "esc"} {
		m := ask()
		if cmd := on(m, k); cmd != nil || m.confirmDelete || !m.toast.empty() {
			t.Fatalf("%q did not cancel the delete", k)
		}
	}
	m := ask()
	if cmd := on(m, "x", "j"); cmd != nil || !m.confirmDelete || currentGroup(m) != "db" {
		t.Fatal("other keys leaked out of the delete question")
	}
}

func TestGroupsConnect(t *testing.T) {
	t.Run("all hosts of the group", func(t *testing.T) {
		m := groupsScreen(nil)
		m.toast = toast{text: "old", level: toastInfo}
		cmd := on(m, "G", "C")
		if !isQuit(cmd) || !slices.Contains(m.execCmd, "w1") || !m.toast.empty() {
			t.Fatalf("quit=%v exec=%v toast=%q, want exec ssh w1", isQuit(cmd), m.execCmd, m.toast.text)
		}
	})
	t.Run("an empty group", func(t *testing.T) {
		m := groupsScreen(nil)
		on(m, "j", "C")
		wantToast(t, "groups", m.toast, "group has no hosts", toastWarn)
	})
	t.Run("a group whose row is stale", func(t *testing.T) {
		m := groupsScreen(nil)
		m.opts.Inventory.Groups = m.opts.Inventory.Groups[:1]
		on(m, "C")
		wantToast(t, "groups", m.toast, "invalid group", toastErr)
	})
	t.Run("above the threshold asks first", func(t *testing.T) {
		m := groupsScreen(func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 1 })
		on(m, "C")
		if !m.confirmConnect || m.confirmConnectCount != 2 {
			t.Fatalf("confirm=%v count=%d, want a confirm for 2 hosts", m.confirmConnect, m.confirmConnectCount)
		}
		if cmd := on(m, "n"); cmd != nil || m.confirmConnect {
			t.Fatal("n did not cancel the connect")
		}
	})
	t.Run("one window needs tmux", func(t *testing.T) {
		m := groupsScreen(nil)
		on(m, "o")
		wantToast(t, "groups", m.toast, "requires an active tmux session", toastWarn)
	})
	t.Run("with a remote command", func(t *testing.T) {
		m := groupsScreen(nil)
		on(m, "G", "ctrl+o")
		if !m.cmdPrompt || m.cmdPromptCrumb != "Groups > web" {
			t.Fatalf("prompt=%v crumb=%q, want the prompt for web", m.cmdPrompt, m.cmdPromptCrumb)
		}
		on(m, "j", "q") // typed into the prompt, not the screen
		if m.cmdInput.Value() != "jq" || currentGroup(m) != "web" || m.quitting {
			t.Fatalf("prompt %q, group %q, quitting %v; want keys typed into the prompt", m.cmdInput.Value(), currentGroup(m), m.quitting)
		}
		cmd := on(m, "enter")
		if !isQuit(cmd) || !strings.Contains(strings.Join(m.execCmd, " "), "jq") {
			t.Fatalf("quit=%v exec=%v, want ssh w1 running jq", isQuit(cmd), m.execCmd)
		}
	})
	t.Run("esc closes the prompt", func(t *testing.T) {
		m := groupsScreen(nil)
		on(m, "ctrl+o", "esc")
		if m.cmdPrompt {
			t.Fatal("esc did not close the prompt")
		}
	})
}

func TestGroupsQuit(t *testing.T) {
	m := groupsScreen(nil)
	if cmd := on(m, "q"); !isQuit(cmd) || !m.quitting {
		t.Fatal("q did not quit")
	}

	m = groupsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	if cmd := on(m, "q"); cmd != nil || !m.confirmQuit {
		t.Fatal("q did not ask to quit")
	}
	wantToast(t, "groups", m.toast, "quit? (y/n)", toastWarn)
	if on(m, "n"); m.confirmQuit || m.quitting || !m.toast.empty() {
		t.Fatal("n did not cancel the quit")
	}
	if cmd := on(m, "q", "y"); !isQuit(cmd) || !m.quitting {
		t.Fatal("y did not quit")
	}
}

func TestGroupsHelp(t *testing.T) {
	m := groupsScreen(nil)
	on(m, "?")
	if !strings.Contains(m.View(), "Editing") {
		t.Fatal("? did not open help")
	}
	if on(m, "j", "q", "d"); !strings.Contains(m.View(), "Editing") || m.quitting || m.confirmDelete {
		t.Fatal("keys in help leaked to the screen")
	}
	if on(m, "?"); strings.Contains(m.View(), "Editing") {
		t.Fatal("? did not close help")
	}
	if on(m, "?", "esc"); strings.Contains(m.View(), "Editing") {
		t.Fatal("esc did not close help")
	}

	unsized := newGroupsModel(Options{Config: config.DefaultConfig()})
	if on(unsized, "?"); !unsized.showHelp || unsized.helpVP.Width != 0 {
		t.Fatal("help before any window size: want it on, with no viewport yet")
	}
}

func TestGroupsSearch(t *testing.T) {
	m := groupsScreen(nil)
	on(m, "/", "w", "e")
	if got := shownGroups(m); !slices.Equal(got, []string{"web"}) {
		t.Fatalf("search we shows %v, want [web]", got)
	}
	if on(m, "enter"); m.focus != focusList || m.search.Value() != "we" {
		t.Fatalf("enter: focus %v query %q, want the list with the query kept", m.focus, m.search.Value())
	}
	if on(m, "esc"); m.search.Value() != "" || len(shownGroups(m)) != 3 {
		t.Fatalf("esc in the list: query %q, want it cleared", m.search.Value())
	}
	if on(m, "/", "z", "z", "enter"); m.search.Value() != "" || len(shownGroups(m)) != 3 {
		t.Fatalf("enter on no match: query %q, want it dropped", m.search.Value())
	}
	if on(m, "tab"); m.focus != focusSearch {
		t.Fatal("tab did not focus search")
	}
	if on(m, "esc"); m.focus != focusList {
		t.Fatal("esc in search did not blur it")
	}
}

func TestGroupsVimNav(t *testing.T) {
	m := groupsScreen(nil)
	if on(m, "G"); currentGroup(m) != "web" {
		t.Fatalf("G: on %q, want web", currentGroup(m))
	}
	if on(m, "g", "g"); currentGroup(m) != "db" {
		t.Fatalf("gg: on %q, want db", currentGroup(m))
	}
	if on(m, "G", "g", "tab", "tab", "g"); currentGroup(m) != "web" {
		t.Fatalf("g, tab, tab, g: on %q, want web (one g is not gg)", currentGroup(m))
	}
}

func TestGroupsToastMsg(t *testing.T) {
	m := groupsScreen(nil)
	if _, cmd := m.Update(toastMsg{text: "opened", level: toastOK}); cmd != nil || m.toast.text != "opened" {
		t.Fatal("toast not shown")
	}
	m = groupsScreen(func(o *Options) { o.Popup = true })
	if _, cmd := m.Update(toastMsg{text: "failed", level: toastErr}); cmd != nil || m.quitting {
		t.Fatal("popup quit on an error")
	}
	if _, cmd := m.Update(toastMsg{text: "opened", level: toastOK}); !isQuit(cmd) || !m.quitting {
		t.Fatal("popup did not quit after a connect")
	}
}

func TestGroupsLayout(t *testing.T) {
	m := groupsScreen(nil)
	if w, h := m.list.Width(), m.list.Height(); w != 98 || h != tabBoxListContentHeight(100, 30) {
		t.Fatalf("list %dx%d, want 98x%d", w, h, tabBoxListContentHeight(100, 30))
	}
	if lines := strings.Split(m.View(), "\n"); len(lines) != 30 {
		t.Fatalf("view is %d lines, want 30", len(lines))
	}
}

func TestGroupsCopyStaleRow(t *testing.T) {
	m := groupsScreen(nil)
	m.opts.Inventory.Groups = m.opts.Inventory.Groups[:1] // db's row now points past the end
	if cmd := on(m, "y"); cmd != nil {
		t.Fatal("copy of a stale row opened a form")
	}
	wantToast(t, "groups", m.toast, "no group selected", toastWarn)
}
