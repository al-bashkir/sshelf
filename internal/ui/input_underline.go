package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// underlineFill is the character used to fill the trailing space of form inputs.
var underlineFill = lipgloss.NewStyle().Foreground(cFrameBorder)

func underlineInput(in textinput.Model, width int) string {
	s := strings.TrimRight(in.View(), "\n")
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	fill := underlineFill.Render(strings.Repeat("─", pad))
	return s + fill
}

func underlineText(s string, focused bool, width int) string {
	s = strings.TrimRight(s, "\n")
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		if focused {
			return checkedStyle.Render(s)
		}
		return s
	}
	fill := underlineFill.Render(strings.Repeat("─", pad))
	if focused {
		return checkedStyle.Render(s) + fill
	}
	return s + fill
}
