package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// These pin the key handling every list screen shares, on the Hosts screen.

func TestQuitQuestionTakesEveryKey(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Config.Defaults.ConfirmQuit = true })
	on(m, "q", "j")
	if h := m.currentHost(); h != "alpha" || m.toast.text != "quit? (y/n)" {
		t.Fatalf("j under the quit question: host %q toast %q, want alpha and the question", h, m.toast.text)
	}
}

func TestConnectQuestionTakesEveryKey(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Config.Defaults.ConnectConfirmThreshold = 2 })
	on(m, "ctrl+a", "enter", "j")
	if h := m.currentHost(); h != "alpha" {
		t.Fatalf("j under the connect question moved to %q", h)
	}
	screenShows(t, m, []string{"Connect 3 hosts?"})
	on(m, "y")
	screenShows(t, m, nil, "Connect 3 hosts?")
}

func TestConnectAsksOnlyAboveTheThreshold(t *testing.T) {
	seven := []string{"h1", "h2", "h3", "h4", "h5", "h6", "h7"}
	for _, c := range []struct {
		threshold int
		hosts     []string
		asks      bool
	}{
		{3, nil, false},   // 3 hosts, not more than 3
		{0, seven, false}, // 0 never asks
		{-1, nil, false},  // unset: 5
		{-1, seven, true},
	} {
		m := hostsScreen(func(o *Options) {
			o.Config.Defaults.ConnectConfirmThreshold = c.threshold
			if c.hosts != nil {
				o.Hosts = c.hosts
			}
		})
		on(m, "ctrl+a", "enter")
		if asks := strings.Contains(plainView(m), "hosts?"); asks != c.asks {
			t.Errorf("threshold %d, %d hosts: asks %v, want %v", c.threshold, len(m.allHosts), asks, c.asks)
		}
	}
}

func TestConnectGoesToTheSelectedHost(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "space", "j", "j")
	screenShows(t, m, []string{"◼"})
	if cmd := on(m, "enter"); !isQuit(cmd) || !strings.Contains(strings.Join(m.execCmd, " "), "alpha") {
		t.Fatalf("exec %v, want ssh to the selected alpha, not the current %q", m.execCmd, m.currentHost())
	}
	on(m, "ctrl+d")
	screenShows(t, m, nil, "◼")
}

func TestCommandPromptClosesOnEnter(t *testing.T) {
	m := hostsScreen(nil)
	if on(m, "ctrl+o", "enter"); m.cmdPrompt {
		t.Fatal("enter on an empty prompt left it open")
	}
	if on(m, "ctrl+o", "u", "p", "enter"); m.cmdPrompt {
		t.Fatal("enter left the prompt open")
	}
}

func TestCommandPromptInputFitsTheModal(t *testing.T) {
	for _, c := range []struct{ w, want int }{
		{100, 70}, // never wider than 70
		{40, 25},
		{12, 1}, // never under 1
	} {
		m := hostsScreen(nil)
		m.Update(tea.WindowSizeMsg{Width: c.w, Height: 30})
		on(m, "ctrl+o")
		if m.cmdInput.Width != c.want {
			t.Fatalf("width %d: input %d wide, want %d", c.w, m.cmdInput.Width, c.want)
		}
	}
}

func TestEscClearsTheSearchAndLeavesIt(t *testing.T) {
	m := hostsScreen(nil)
	on(m, "/", "a", "esc", "j")
	if q, h := m.search.Value(), m.currentHost(); q != "" || h != "beta" {
		t.Fatalf("after esc: search %q host %q, want no search and j moving to beta", q, h)
	}
}

func TestVimNavStaysOnThePage(t *testing.T) {
	hs := make([]string, 10)
	for i := range hs {
		hs[i] = fmt.Sprintf("h%02d", i)
	}
	m := hostsScreen(func(o *Options) { o.Hosts = hs })
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 12}) // 4 rows a page
	on(m, "right", "j")                                 // page 2 (h04-h07), on h05
	for _, c := range []struct{ key, want string }{
		{"g", "h05"}, // one g waits for the second
		{"g", "h04"}, // gg: top of the page
		{"G", "h07"}, // bottom of the page
	} {
		if on(m, c.key); m.currentHost() != c.want {
			t.Fatalf("%s: on %q, want %q", c.key, m.currentHost(), c.want)
		}
	}
}

func TestHostsUnhideAListedHiddenHost(t *testing.T) {
	m := hostsScreen(func(o *Options) { o.Inventory = config.Inventory{HiddenHosts: []string{"alpha"}} })
	if got := msgOf(on(m, "H", "up", "ctrl+h")); got != (toggleHiddenHostMsg{host: "alpha", hide: false}) {
		t.Fatalf("ctrl+h on hidden alpha: %#v, want an unhide", got)
	}
}
