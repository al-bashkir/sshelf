package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHexToANSIBG(t *testing.T) {
	for in, want := range map[string]string{
		"#1e1e2e": "\x1b[48;2;30;30;46m",
		"282A36":  "\x1b[48;2;40;42;54m",
		"#ffffff": "\x1b[48;2;255;255;255m",
		"":        "",
		"#fff":    "",
		"#1e1e2":  "",
		"#zz1e2e": "",
		"#1ezz2e": "",
		"#1e1ezz": "",
	} {
		if got := hexToANSIBG(in); got != want {
			t.Errorf("hexToANSIBG(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEveryThemeBackgroundParses(t *testing.T) {
	for name, cs := range builtinColorSchemes {
		if (cs.Background != "") != (hexToANSIBG(cs.Background) != "") {
			t.Errorf("%s: background %q does not parse", name, cs.Background)
		}
	}
}

func TestSetAccentColor(t *testing.T) {
	t.Cleanup(func() { ApplyColorScheme("", "") })
	custom := lipgloss.AdaptiveColor{Light: "#ff0000", Dark: "#ff0000"}
	for name, want := range map[string]lipgloss.AdaptiveColor{
		"":          defaultAccent,
		" Default ": defaultAccent,
		"Cyan":      accentPresets["cyan"],
		"#FF0000":   custom, // any lipgloss color, lower-cased
	} {
		SetAccentColor(name)
		if cAccent != want || cFocusedBorder != want {
			t.Errorf("SetAccentColor(%q): accent %v, focused border %v, want %v", name, cAccent, cFocusedBorder, want)
		}
	}
}

func TestWithThemeBG(t *testing.T) {
	t.Cleanup(func() { ApplyColorScheme("", "") })
	s := "a\x1b[0mb\x1b[mc"
	ApplyColorScheme("", "")
	if got := withThemeBG(s); got != s {
		t.Fatalf("no theme: got %q, want %q unchanged", got, s)
	}
	ApplyColorScheme("nord", "")
	bg := "\x1b[48;2;46;52;64m" // nord0, #2e3440
	if got, want := withThemeBG(s), bg+"a\x1b[0m"+bg+"b\x1b[m"+bg+"c"; got != want {
		t.Fatalf("nord: got %q, want %q", got, want)
	}
}
