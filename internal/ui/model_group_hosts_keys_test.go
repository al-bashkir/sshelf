package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/al-bashkir/sshelf/internal/config"
)

// groupHostsScreen opens group index (0: prod with alpha, beta, gamma and
// user ops; 1: empty) in a 100x30 terminal. alpha has a host config. The
// cursor starts on the first host. Connects never use tmux.
func groupHostsScreen(index int, edit func(*Options)) *groupHostsModel {
	o := Options{Config: config.DefaultConfig(), NoTmux: true, Inventory: config.Inventory{
		Hosts: []config.Host{{Host: "alpha", User: "a"}},
		Groups: []config.Group{
			{Name: "prod", User: "ops", Hosts: []string{"alpha", "beta", "gamma"}},
			{Name: "empty"},
		},
	}}
	if edit != nil {
		edit(&o)
	}
	m := newGroupHostsModel(o, index)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

func TestGroupHostsKeysOpenOtherScreens(t *testing.T) {
	m0 := groupHostsScreen(0, nil)
	alphaCopy := config.Host{Host: suggestCopyHostKey(m0.opts.Inventory, "alpha"), User: "a"}
	cases := []struct {
		key  tea.KeyMsg
		want tea.Msg
	}{
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1"), Alt: true}, switchScreenMsg{to: screenHosts}},
		{hk("alt+2"), switchScreenMsg{to: screenGroups}},
		{hk("alt+3"), openDefaultsFormMsg{returnTo: screenGroupHosts}},
		{hk("a"), openHostPickerMsg{groupIndex: 0, returnTo: screenGroupHosts}},
		{hk("A"), openHostPickerMsg{groupIndex: 0, returnTo: screenGroupHosts}},
		{hk("c"), openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: 0}},
		{hk("e"), openHostFormMsg{host: "alpha", returnTo: screenGroupHosts}},
		{hk("y"), openHostFormPrefillMsg{host: alphaCopy, returnTo: screenGroupHosts}},
	}
	for _, c := range cases {
		m := groupHostsScreen(0, nil)
		_, cmd := m.Update(c.key)
		if got := msgOf(cmd); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%q gave %#v, want %#v", c.key.String(), got, c.want)
		}
	}
}

func TestGroupHostsKeysWithoutHost(t *testing.T) {
	for _, k := range []string{"e", "y", "d", "enter", "o", "O"} {
		m := groupHostsScreen(1, nil)
		if cmd := on(m, k); cmd != nil || m.confirmRemove || m.toast.text != "no host selected" {
			t.Fatalf("%q with no host: toast %q remove=%v, want no host selected", k, m.toast.text, m.confirmRemove)
		}
	}
	m := groupHostsScreen(1, nil)
	if cmd := on(m, "ctrl+o"); cmd != nil || m.cmdPrompt || !m.toast.empty() {
		t.Fatal("ctrl+o with no host did something")
	}
	m = groupHostsScreen(0, nil)
	if cmd := on(m, "j", "y"); cmd != nil || m.toast.text != "no host config" {
		t.Fatalf("copy of beta: toast %q, want no host config", m.toast.text)
	}
}

func TestGroupHostsListKeysTypeInSearch(t *testing.T) {
	for _, k := range []string{"d", "a", "c", "e", "y", "o", "O", "q"} {
		m := groupHostsScreen(0, nil)
		on(m, "/", k)
		if m.search.Value() != k || m.confirmRemove || len(m.execCmd) != 0 || !m.toast.empty() {
			t.Fatalf("%q in search: query %q, remove=%v toast=%q; want it typed", k, m.search.Value(), m.confirmRemove, m.toast.text)
		}
	}
	m := groupHostsScreen(0, nil)
	if on(m, "/", "ctrl+o", "ctrl+a"); m.cmdPrompt || len(m.selectedHosts()) != 0 {
		t.Fatal("list keys acted from the search box")
	}
}

func TestGroupHostsRemove(t *testing.T) {
	ask := func(keys ...string) *groupHostsModel {
		m := groupHostsScreen(0, nil)
		if cmd := on(m, append(keys, "d")...); cmd != nil || !m.confirmRemove {
			t.Fatal("d did not ask")
		}
		return m
	}
	m := ask()
	wantToast(t, "group hosts", m.toast, "remove 1? (y/n)", toastWarn)

	for _, k := range []string{"y", "Y", "enter"} {
		m := ask()
		got := msgOf(on(m, k))
		if !reflect.DeepEqual(got, removeHostsMsg{groupIndex: 0, hosts: []string{"alpha"}}) || m.confirmRemove || m.removeList != nil {
			t.Fatalf("%q gave %#v, want remove of alpha", k, got)
		}
	}
	for _, k := range []string{"n", "N", "esc"} {
		m := ask()
		if cmd := on(m, k); cmd != nil || m.confirmRemove || m.removeList != nil || !m.toast.empty() {
			t.Fatalf("%q did not cancel the remove", k)
		}
	}
	m = ask()
	if cmd := on(m, "x", "j"); cmd != nil || !m.confirmRemove || m.currentHost() != "alpha" {
		t.Fatal("other keys leaked out of the remove question")
	}

	m = ask("space", "j", "space")
	wantToast(t, "group hosts", m.toast, "remove 2? (y/n)", toastWarn)
	if got := msgOf(on(m, "y")); !reflect.DeepEqual(got, removeHostsMsg{groupIndex: 0, hosts: []string{"alpha", "beta"}}) {
		t.Fatalf("y gave %#v, want remove of alpha and beta", got)
	}
}

func TestGroupHostsSelection(t *testing.T) {
	m := groupHostsScreen(0, nil)
	if on(m, "space"); !slices.Equal(m.selectedHosts(), []string{"alpha"}) {
		t.Fatalf("space: selected %v, want [alpha]", m.selectedHosts())
	}
	if on(m, "ctrl+a"); len(m.selectedHosts()) != 3 {
		t.Fatalf("ctrl+a: selected %v, want all", m.selectedHosts())
	}
	if on(m, "ctrl+d"); len(m.selectedHosts()) != 0 {
		t.Fatalf("ctrl+d: selected %v, want none", m.selectedHosts())
	}
}

func TestGroupHostsConnect(t *testing.T) {
	for _, k := range []string{"enter", "O"} {
		m := groupHostsScreen(0, nil)
		m.toast = toast{text: "old", level: toastInfo}
		cmd := on(m, k)
		if !isQuit(cmd) || !strings.Contains(strings.Join(m.execCmd, " "), "alpha") || !m.toast.empty() {
			t.Fatalf("%q: quit=%v exec=%v toast=%q, want exec ssh alpha", k, isQuit(cmd), m.execCmd, m.toast.text)
		}
	}
	t.Run("one window needs tmux", func(t *testing.T) {
		m := groupHostsScreen(0, nil)
		on(m, "o")
		wantToast(t, "group hosts", m.toast, "requires an active tmux session", toastWarn)
	})
	t.Run("above the threshold asks first", func(t *testing.T) {
		m := groupHostsScreen(0, func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 1 })
		on(m, "ctrl+a", "enter")
		if !m.confirmConnect || m.confirmConnectCount != 3 {
			t.Fatalf("confirm=%v count=%d, want a confirm for 3 hosts", m.confirmConnect, m.confirmConnectCount)
		}
		if cmd := on(m, "n"); cmd != nil || m.confirmConnect {
			t.Fatal("n did not cancel the connect")
		}
	})
	t.Run("with a remote command", func(t *testing.T) {
		m := groupHostsScreen(0, nil)
		on(m, "ctrl+o")
		if want := cmdPromptCrumb("Groups > prod", []string{"alpha"}); !m.cmdPrompt || m.cmdPromptCrumb != want {
			t.Fatalf("prompt=%v crumb=%q, want %q", m.cmdPrompt, m.cmdPromptCrumb, want)
		}
		on(m, "j", "q")
		if m.cmdInput.Value() != "jq" || m.currentHost() != "alpha" {
			t.Fatalf("prompt %q on %q, want keys typed into the prompt", m.cmdInput.Value(), m.currentHost())
		}
		if cmd := on(m, "enter"); !isQuit(cmd) || !strings.Contains(strings.Join(m.execCmd, " "), "jq") {
			t.Fatalf("exec %v, want ssh alpha running jq", m.execCmd)
		}
	})
}

func TestGroupHostsEscAndQGoBackLast(t *testing.T) {
	for _, k := range []string{"esc", "q"} {
		m := groupHostsScreen(0, nil)
		if cmd := on(m, "space", k); cmd != nil || len(m.selectedHosts()) != 0 {
			t.Fatalf("%q with a selection: want it cleared first", k)
		}
		if cmd := on(m, "/", "b", "enter", k); cmd != nil || m.search.Value() != "" {
			t.Fatalf("%q with a query: query %q, want it cleared first", k, m.search.Value())
		}
		if got := msgOf(on(m, k)); !reflect.DeepEqual(got, switchScreenMsg{to: screenGroups}) {
			t.Fatalf("%q with nothing to clear gave %#v, want back to Groups", k, got)
		}
	}
	m := groupHostsScreen(0, nil)
	if cmd := on(m, "/", "esc"); cmd != nil || m.focus != focusList {
		t.Fatal("esc in an empty search box did not blur it")
	}
}

func TestGroupHostsHelp(t *testing.T) {
	m := groupHostsScreen(0, nil)
	on(m, "?")
	if !strings.Contains(m.View(), "Selection") {
		t.Fatal("? did not open help")
	}
	if on(m, "j", "q", "d"); !strings.Contains(m.View(), "Selection") || m.confirmRemove {
		t.Fatal("keys in help leaked to the screen")
	}
	if on(m, "?"); strings.Contains(m.View(), "Selection") {
		t.Fatal("? did not close help")
	}
	if on(m, "?", "esc"); strings.Contains(m.View(), "Selection") {
		t.Fatal("esc did not close help")
	}
}

func TestGroupHostsSearch(t *testing.T) {
	m := groupHostsScreen(0, nil)
	on(m, "/", "b", "e")
	if got := rowHosts(m.l.Items()); !slices.Equal(got, []string{"beta"}) {
		t.Fatalf("search be shows %v, want [beta]", got)
	}
	if on(m, "enter"); m.focus != focusList || m.search.Value() != "be" {
		t.Fatalf("enter: focus %v query %q, want the list with the query kept", m.focus, m.search.Value())
	}
	if on(m, "tab", "z", "z", "enter"); m.search.Value() != "" {
		t.Fatalf("enter on no match: query %q, want it dropped", m.search.Value())
	}
}

func TestGroupHostsVimNav(t *testing.T) {
	m := groupHostsScreen(0, nil)
	if on(m, "G"); m.currentHost() != "gamma" {
		t.Fatalf("G: on %q, want gamma", m.currentHost())
	}
	if on(m, "g", "g"); m.currentHost() != "alpha" {
		t.Fatalf("gg: on %q, want alpha", m.currentHost())
	}
	if on(m, "G", "g", "tab", "tab", "g"); m.currentHost() != "gamma" {
		t.Fatalf("g, tab, tab, g: on %q, want gamma (one g is not gg)", m.currentHost())
	}
}

func TestGroupHostsToastMsg(t *testing.T) {
	m := groupHostsScreen(0, nil)
	if _, cmd := m.Update(toastMsg{text: "opened", level: toastOK}); cmd != nil || m.toast.text != "opened" {
		t.Fatal("toast not shown")
	}
	m = groupHostsScreen(0, func(o *Options) { o.Popup = true })
	if _, cmd := m.Update(toastMsg{text: "failed", level: toastErr}); cmd != nil || m.quitting {
		t.Fatal("popup quit on an error")
	}
	if _, cmd := m.Update(toastMsg{text: "opened", level: toastOK}); !isQuit(cmd) || !m.quitting {
		t.Fatal("popup did not quit after a connect")
	}
}

func TestGroupHostsLayout(t *testing.T) {
	m := groupHostsScreen(0, nil)
	if w, h := m.list.Width(), m.list.Height(); w != 98 || h != tabBoxListContentHeight(100, 30) {
		t.Fatalf("list %dx%d, want 98x%d", w, h, tabBoxListContentHeight(100, 30))
	}
	if lines := strings.Split(m.View(), "\n"); len(lines) != 30 {
		t.Fatalf("view is %d lines, want 30", len(lines))
	}
}
