package ui

import (
	"slices"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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
	screenShows(t, hostsHelp(100, 80),
		[]string{"Hosts keybindings", "  Navigation", "q  quit", "Esc/? close  j/k scroll"}, "%")
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
