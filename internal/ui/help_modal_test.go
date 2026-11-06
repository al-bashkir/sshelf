package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// hostsHelp opens the Hosts help (39 lines) in a w x h window.
func hostsHelp(w, h int) *hostsModel {
	m := hostsScreen(nil)
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	on(m, "?")
	return m
}

func TestHelpSectionsSkipDisabledKeys(t *testing.T) {
	kept := key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "kept"))
	gone := key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "gone"), key.WithDisabled())
	got := ansi.Strip(renderHelpSections([]helpSection{
		{title: "A", keys: []key.Binding{gone, kept}},
		{title: "B", keys: []key.Binding{kept}},
	}))
	if want := "A\n  y  kept\n\nB\n  y  kept"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestHelpShowsEveryKeyWhenItFits(t *testing.T) {
	for _, h := range []int{80, 43} { // 43: 39 lines, border and padding
		screenShows(t, hostsHelp(100, h),
			[]string{"Hosts keybindings", "  Navigation", "q  quit", "Esc/? close  j/k scroll"}, "%")
	}
}

func TestHelpScrolls(t *testing.T) {
	m := hostsHelp(100, 12)
	screenShows(t, m, []string{"Hosts keybindings", "▼ 0%"}, "▲", "Esc/? close")
	for _, k := range [][2]string{{"j", "k"}, {"down", "up"}, {"ctrl+d", "ctrl+u"}} {
		on(m, k[0])
		screenShows(t, m, []string{"▲ ▼ "}, "Hosts keybindings")
		on(m, k[1])
		screenShows(t, m, []string{"Hosts keybindings", "▼ 0%"}, "▲")
	}
	on(m, "j")
	screenShows(t, m, []string{"▲ ▼ 3%"})
	on(m, slices.Repeat([]string{"j"}, 40)...)
	screenShows(t, m, []string{"▲ 100%", "Esc/? close  j/k scroll"}, "▼")
	on(m, "k")
	screenShows(t, m, []string{"▲ ▼ 96%"})
}

func TestHelpBeforeTheWindowSize(t *testing.T) {
	m := newHostsModel(Options{Config: config.DefaultConfig(), Hosts: []string{"alpha"}})
	on(m, "?")
	screenShows(t, m, []string{"Hosts\n\nNavigation", "q  quit"}, "keybindings")
}

func TestHelpFitsTheWindow(t *testing.T) {
	for _, s := range [][2]int{{100, 80}, {100, 43}, {100, 42}, {100, 30}, {100, 12}, {32, 12}, {31, 12}, {26, 12}, {20, 12}} {
		lines := strings.Split(plainView(hostsHelp(s[0], s[1])), "\n")
		if len(lines) > s[1] {
			t.Errorf("%dx%d: help is %d rows tall", s[0], s[1], len(lines))
		}
		for _, l := range lines {
			if w := lipgloss.Width(l); w > s[0] {
				t.Errorf("%dx%d: help is %d columns wide", s[0], s[1], w)
				break
			}
		}
	}
}
