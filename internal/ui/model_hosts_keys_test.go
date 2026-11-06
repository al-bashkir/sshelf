package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"
)

// hk builds the key a terminal sends for s.
func hk(s string) tea.KeyMsg {
	special := map[string]tea.KeyMsg{
		"enter":     {Type: tea.KeyEnter},
		"esc":       {Type: tea.KeyEsc},
		"tab":       {Type: tea.KeyTab},
		"up":        {Type: tea.KeyUp},
		"down":      {Type: tea.KeyDown},
		"right":     {Type: tea.KeyRight},
		"shift+tab": {Type: tea.KeyShiftTab},
		"backspace": {Type: tea.KeyBackspace},
		"space":     {Type: tea.KeySpace, Runes: []rune{' '}},
		"ctrl+a":    {Type: tea.KeyCtrlA},
		"ctrl+d":    {Type: tea.KeyCtrlD},
		"ctrl+f":    {Type: tea.KeyCtrlF},
		"ctrl+h":    {Type: tea.KeyCtrlH},
		"ctrl+o":    {Type: tea.KeyCtrlO},
		"ctrl+s":    {Type: tea.KeyCtrlS},
		"ctrl+u":    {Type: tea.KeyCtrlU},
		"alt+1":     {Type: tea.KeyRunes, Runes: []rune("1"), Alt: true},
		"alt+2":     {Type: tea.KeyRunes, Runes: []rune("2"), Alt: true},
		"alt+3":     {Type: tea.KeyRunes, Runes: []rune("3"), Alt: true},
	}
	if k, ok := special[s]; ok {
		return k
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// hostsScreen opens the hosts tab on alpha, beta, gamma in a 100x30 terminal.
func hostsScreen(edit func(*Options)) *hostsModel {
	o := Options{Config: config.DefaultConfig(), Hosts: []string{"alpha", "beta", "gamma"}}
	o.Config.Defaults.ConfirmQuit = false
	if edit != nil {
		edit(&o)
	}
	m := newHostsModel(o)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

// on sends the keys and returns the command of the last one.
func on(m tea.Model, ks ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range ks {
		_, cmd = m.Update(hk(k))
	}
	return cmd
}

func msgOf(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func isQuit(cmd tea.Cmd) bool {
	_, ok := msgOf(cmd).(tea.QuitMsg)
	return ok
}

func TestHostsKeysOpenOtherScreens(t *testing.T) {
	inv := config.Inventory{Hosts: []config.Host{{Host: "alpha", User: "u"}}}
	tc := []struct {
		key  string
		want tea.Msg
	}{
		{"alt+2", switchScreenMsg{to: screenGroups}},
		{"alt+3", openDefaultsFormMsg{returnTo: screenHosts}},
		{"a", openGroupPickerMsg{hosts: []string{"alpha"}}},
		{"c", openCustomHostMsg{returnTo: screenHosts, groupIndex: -1}},
		{"e", openHostFormMsg{host: "alpha", returnTo: screenHosts}},
		{"y", openHostFormPrefillMsg{host: config.Host{Host: "alpha-copy", User: "u"}, returnTo: screenHosts}},
		{"ctrl+h", toggleHiddenHostMsg{host: "alpha", hide: true}},
	}
	for _, c := range tc {
		t.Run(c.key, func(t *testing.T) {
			m := hostsScreen(func(o *Options) { o.Inventory = inv })
			if got := msgOf(on(m, c.key)); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v want %#v", got, c.want)
			}
		})
	}
}

func TestHostsKeysWithoutHost(t *testing.T) {
	for _, k := range []string{"a", "e", "y", "enter", "O"} {
		m := hostsScreen(func(o *Options) { o.Hosts = nil })
		if cmd := on(m, k); cmd != nil || m.toast.text != "no host selected" {
			t.Fatalf("%s: cmd=%v toast=%q, want no command and \"no host selected\"", k, cmd != nil, m.toast.text)
		}
	}
	m := hostsScreen(func(o *Options) { o.Hosts = nil })
	if on(m, "ctrl+h") != nil || on(m, "ctrl+o") != nil || m.cmdPrompt {
		t.Fatal("hide or command prompt acted without a host")
	}
	if m := hostsScreen(nil); on(m, "y") != nil || m.toast.text != "no host config" {
		t.Fatalf("copy without config: toast %q", m.toast.text)
	}
}

func TestHostsConnect(t *testing.T) {
	m := hostsScreen(nil)
	if cmd := on(m, "enter"); !isQuit(cmd) || m.execCmd[len(m.execCmd)-1] != "alpha" {
		t.Fatalf("enter: quit=%v exec=%q, want ssh to alpha", isQuit(cmd), m.execCmd)
	}
	m = hostsScreen(nil)
	if cmd := on(m, "down", "O"); !isQuit(cmd) || m.execCmd[len(m.execCmd)-1] != "beta" {
		t.Fatalf("O: quit=%v exec=%q, want ssh to beta", isQuit(cmd), m.execCmd)
	}
	m = hostsScreen(nil)
	if on(m, "o") != nil || m.toast.text != "requires an active tmux session" {
		t.Fatalf("o: toast %q", m.toast.text)
	}
	m = hostsScreen(nil)
	if on(m, "space", "down", "space", "O") != nil || !strings.HasPrefix(m.toast.text, "select single host") {
		t.Fatalf("O with two: toast %q", m.toast.text)
	}
	if isQuit(on(m, "enter")) || !strings.Contains(m.toast.text, "multi-host") {
		t.Fatalf("enter with two outside tmux: toast %q", m.toast.text)
	}
}

func TestHostsSelection(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "space")
	if got := m.selectedHosts(); !slices.Equal(got, []string{"alpha"}) {
		t.Fatalf("space: %v", got)
	}
	on(m, "ctrl+a")
	if got := m.selectedHosts(); len(got) != 3 {
		t.Fatalf("ctrl+a: %v", got)
	}
	on(m, "ctrl+d")
	if got := m.selectedHosts(); len(got) != 0 {
		t.Fatalf("ctrl+d: %v", got)
	}
	on(m, "space", "esc") // esc clears the selection first
	if got := m.selectedHosts(); len(got) != 0 {
		t.Fatalf("esc: %v", got)
	}
}

func TestHostsConfirmConnect(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Hosts = []string{"h1", "h2", "h3", "h4", "h5", "h6", "h7"} })
	on(m, "ctrl+a", "enter")
	if !strings.Contains(m.View(), "Connect 7 hosts?") {
		t.Fatal("no confirm box for 7 hosts")
	}
	on(m, "x")
	if !strings.Contains(m.View(), "Connect 7 hosts?") {
		t.Fatal("other key closed the confirm box")
	}
	if on(m, "n"); strings.Contains(m.View(), "Connect 7 hosts?") {
		t.Fatal("n did not close the confirm box")
	}
	on(m, "enter", "y")
	if !strings.Contains(m.toast.text, "multi-host") {
		t.Fatalf("y: toast %q, want the multi-host error", m.toast.text)
	}
}

func TestHostsQuit(t *testing.T) {
	m := hostsScreen(nil)
	if cmd := on(m, "q"); !isQuit(cmd) || !m.quitting {
		t.Fatal("q did not quit")
	}

	m = hostsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	if cmd := on(m, "q"); cmd != nil || m.toast.text != "quit? (y/n)" {
		t.Fatalf("q with confirm: toast %q", m.toast.text)
	}
	if cmd := on(m, "n"); isQuit(cmd) {
		t.Fatal("n quit")
	}
	if cmd := on(m, "q", "y"); !isQuit(cmd) {
		t.Fatal("q, y did not quit")
	}

	m = hostsScreen(nil)
	if cmd := on(m, "/", "q"); isQuit(cmd) || m.search.Value() != "q" {
		t.Fatalf("q in search: quit=%v search=%q", isQuit(cmd), m.search.Value())
	}
}

func TestHostsHelp(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "?")
	if !strings.Contains(m.View(), "Selection") {
		t.Fatal("? did not open help")
	}
	if on(m, "j", "q"); !strings.Contains(m.View(), "Selection") || m.quitting {
		t.Fatal("keys in help leaked to the screen")
	}
	if on(m, "?"); strings.Contains(m.View(), "Selection") {
		t.Fatal("? did not close help")
	}
	if on(m, "?", "esc"); strings.Contains(m.View(), "Selection") {
		t.Fatal("esc did not close help")
	}
}

func TestHostsSearch(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "/", "g", "a", "m")
	if v := m.View(); strings.Contains(v, "alpha") || !strings.Contains(v, "gamma") {
		t.Fatalf("filter gam:\n%s", v)
	}
	if got := msgOf(on(m, "enter", "a")); !reflect.DeepEqual(got, openGroupPickerMsg{hosts: []string{"gamma"}}) {
		t.Fatalf("enter accepts the query, then a: %#v", got)
	}
	on(m, "esc") // clears the search
	if v := m.View(); !strings.Contains(v, "alpha") {
		t.Fatalf("esc did not clear the search:\n%s", v)
	}

	// While the search has focus, action keys type into it.
	m = hostsScreen(nil)
	if cmd := on(m, "ctrl+f", "a", "c", "e", "y", "o", "O", "H", "r", "G", "space", "ctrl+h", "ctrl+o"); cmd != nil && msgOf(cmd) != nil {
		t.Fatalf("action ran from search: %#v", msgOf(cmd))
	}
	if got := m.search.Value(); got != "aceyoOHrG" { // ctrl+h is backspace: it deletes the space
		t.Fatalf("search %q", got)
	}
	if m.cmdPrompt || m.showHidden {
		t.Fatal("search keys changed the screen")
	}

	// Selection keys and ctrl+o belong to the search box while it has focus.
	m = hostsScreen(nil)
	on(m, "space", "/", "a", "ctrl+a", "ctrl+d", "ctrl+o")
	if got := m.selectedHosts(); !slices.Equal(got, []string{"alpha"}) || m.cmdPrompt {
		t.Fatalf("search ctrl keys: selection %v prompt=%v, want [alpha] and no prompt", got, m.cmdPrompt)
	}

	m = hostsScreen(nil)
	on(m, "tab", "b", "tab", "space")
	if got := m.selectedHosts(); !slices.Equal(got, []string{"beta"}) {
		t.Fatalf("tab to search, b, tab back, space: %v", got)
	}
	on(m, "/", "esc") // esc in search only leaves it
	if got := m.search.Value(); got != "b" {
		t.Fatalf("esc in search: %q", got)
	}
}

func TestHostsVimNav(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "G")
	if h := m.currentHost(); h != "gamma" {
		t.Fatalf("G: %q", h)
	}
	on(m, "g", "g")
	if h := m.currentHost(); h != "alpha" {
		t.Fatalf("gg: %q", h)
	}
}

func TestHostsReload(t *testing.T) {
	m := hostsScreen(nil)
	if cmd := on(m, "r"); cmd == nil || m.toast.text != "reloading" || !m.reloading {
		t.Fatalf("r: toast %q reloading=%v", m.toast.text, m.reloading)
	}
	m.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: []string{"x1", "x2", "x3", "x4"}, SkippedLines: 2}})
	if m.toast.text != "4 hosts loaded" || m.reloading || m.opts.SkippedLines != 2 || !strings.Contains(m.View(), "x4") {
		t.Fatalf("reloaded: toast %q reloading=%v skipped=%d", m.toast.text, m.reloading, m.opts.SkippedLines)
	}

	// With known_hosts off, r is not bound (the app keeps the binding in step
	// with the setting).
	m = hostsScreen(func(o *Options) { o.Config.Defaults.LoadKnownHosts = false })
	if cmd := on(m, "r"); cmd != nil || m.toast.text != "" || m.reloading {
		t.Fatalf("r disabled: toast %q reloading=%v", m.toast.text, m.reloading)
	}
}

func TestHostsShowHidden(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Inventory = config.Inventory{HiddenHosts: []string{"alpha"}} })
	if strings.Contains(m.View(), "alpha") {
		t.Fatal("hidden host shown")
	}
	if on(m, "H"); !strings.Contains(m.View(), "alpha") {
		t.Fatal("H did not show hidden hosts")
	}
	if on(m, "H"); strings.Contains(m.View(), "alpha") {
		t.Fatal("second H did not hide them")
	}
}

func TestHostsToastMsg(t *testing.T) {
	m := hostsScreen(nil)
	if _, cmd := m.Update(toastMsg{text: "done", level: toastOK}); cmd != nil || m.toast.text != "done" {
		t.Fatalf("toast %q", m.toast.text)
	}
	m = hostsScreen(func(o *Options) { o.Popup = true })
	if _, cmd := m.Update(toastMsg{text: "opened", level: toastOK}); !isQuit(cmd) {
		t.Fatal("popup did not quit after a good toast")
	}
	m = hostsScreen(func(o *Options) { o.Popup = true })
	if _, cmd := m.Update(toastMsg{text: "failed", level: toastErr}); cmd != nil || m.quitting {
		t.Fatal("popup quit after an error")
	}
}

func TestHostsCommandPrompt(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "ctrl+o")
	if !m.cmdPrompt || !strings.Contains(m.cmdPromptCrumb, "alpha") {
		t.Fatalf("ctrl+o: prompt=%v crumb=%q", m.cmdPrompt, m.cmdPromptCrumb)
	}
	if on(m, "esc"); m.cmdPrompt {
		t.Fatal("esc did not close the prompt")
	}
	on(m, "ctrl+o", "u", "p")
	if cmd := on(m, "enter"); !isQuit(cmd) || !strings.Contains(strings.Join(m.execCmd, " "), "up") {
		t.Fatalf("prompt enter: quit=%v exec=%q", isQuit(cmd), m.execCmd)
	}
}

func TestHostsLayout(t *testing.T) {
	long := strings.Repeat("h", 150)
	m := hostsScreen(func(o *Options) {
		o.Hosts = []string{"beta", long} // the long host is not the cursor row
		o.Inventory = config.Inventory{Hosts: []config.Host{{Host: long}}}
	})
	fits := func(what string) {
		t.Helper()
		lines := strings.Split(m.View(), "\n")
		if len(lines) != 30 {
			t.Fatalf("%s: view is %d lines, want 30", what, len(lines))
		}
		for _, l := range lines {
			if w := lipgloss.Width(l); w > 100 {
				t.Fatalf("%s: line %d wide, want <= 100: %q", what, w, l)
			}
		}
	}
	fits("long host")
	if !strings.Contains(m.View(), "⚙") {
		t.Fatal("config badge of a long host is cut off")
	}
	on(m, "/", strings.Repeat("z", 120)+"END")
	fits("long search")
	if !strings.Contains(m.View(), "END") {
		t.Fatal("long search hides its end")
	}
}

// One row per line, the cursor on the current host.
func TestHostsRows(t *testing.T) {
	lines := strings.Split(hostsScreen(nil).View(), "\n")
	row := func(s string) int {
		for i, l := range lines {
			if strings.Contains(l, s) {
				return i
			}
		}
		t.Fatalf("no line has %q", s)
		return -1
	}
	if a, b := row("▸ ◻ alpha"), row("  ◻ beta"); b != a+1 {
		t.Fatalf("alpha on line %d, beta on %d: want adjacent", a, b)
	}
}

func TestHostsQuitFlagsForTheApp(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	on(m, "q", "n")
	if m.toast.text != "" || m.quitting {
		t.Fatalf("n: toast %q quitting=%v, want cleared", m.toast.text, m.quitting)
	}
	on(m, "q", "y")
	if !m.quitting {
		t.Fatal("y: quitting not set")
	}
	m = hostsScreen(func(o *Options) { o.Popup = true })
	m.Update(toastMsg{text: "opened", level: toastOK})
	if !m.quitting {
		t.Fatal("popup: quitting not set")
	}
}

func TestHostsReloadDropsGoneSelections(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "space", "down", "space") // alpha, beta
	m.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: []string{"beta", "gamma"}}})
	if got := m.selectedHosts(); !slices.Equal(got, []string{"beta"}) {
		t.Fatalf("selection %v after reload, want [beta]", got)
	}
}

// A pending g does not survive a trip to the search box.
func TestHostsPendingGResetsOnFocusChange(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "G", "g", "tab", "tab", "g")
	if h := m.currentHost(); h != "gamma" {
		t.Fatalf("current %q, want gamma (one g is not gg)", h)
	}
}
