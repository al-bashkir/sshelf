package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpContent generates the rendered help text body (without the box).
func helpContent(title string, keys helpMap) string {
	header := helpTitleStyle.Render(title + " keybindings")
	body := renderHelpSections(keys.sections)
	footer := styledFooter("Esc/? close  j/k scroll")
	return header + "\n\n" + body + "\n\n" + footer
}

// renderHelpSections renders keybindings grouped by named sections.
func renderHelpSections(sections []helpSection) string {
	keyStyle := lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	sectionStyle := secondaryStyle.Bold(true)

	var sb strings.Builder
	for i, sec := range sections {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(sectionStyle.Render(sec.title))
		sb.WriteByte('\n')
		for _, k := range sec.keys {
			if !k.Enabled() {
				continue
			}
			help := k.Help()
			sb.WriteString("  ")
			sb.WriteString(keyStyle.Render(help.Key))
			sb.WriteString(hintStyle.Render("  " + help.Desc))
			sb.WriteByte('\n')
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func helpBoxWidth(termW int) int {
	boxW := min(helpMaxBoxWidth, termW-confirmDialogMargin)
	if boxW < 30 {
		boxW = min(termW, 30)
	}
	return boxW
}

func helpInnerWidth(boxW int) int {
	// Subtract border (2) + horizontal padding (2*2) = 6.
	innerW := boxW - confirmDialogPadding
	if innerW < 20 {
		innerW = 0
	}
	return innerW
}

// initHelpViewport creates a viewport sized for the help modal and sets its content.
func initHelpViewport(width, height int, title string, keys helpMap) viewport.Model {
	boxW := helpBoxWidth(width)
	innerW := helpInnerWidth(boxW)

	content := helpContent(title, keys)

	// Size viewport to fit content, but cap at available terminal height.
	contentLines := strings.Count(content, "\n") + 1
	maxVPH := height - helpBoxOverhead
	if maxVPH < 3 {
		maxVPH = 3
	}
	vpH := min(contentLines, maxVPH)

	vp := viewport.New(innerW, vpH)
	vp.SetContent(content)
	return vp
}

// updateHelpViewport sends a key message to the viewport for scrolling.
func updateHelpViewport(vp *viewport.Model, msg tea.KeyMsg) {
	s := msg.String()
	switch s {
	case "j", "down":
		vp.ScrollDown(1)
	case "k", "up":
		vp.ScrollUp(1)
	case "pgdown", "ctrl+d":
		vp.HalfPageDown()
	case "pgup", "ctrl+u":
		vp.HalfPageUp()
	}
}

func renderHelpModal(width, height int, title string, keys helpMap) string {
	return renderHelpModalWithVP(width, height, title, keys, nil)
}

func renderHelpModalWithVP(width, height int, title string, keys helpMap, vp *viewport.Model) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Help"
	}

	// Fallback for very early render before we have window size.
	if width <= 0 || height <= 0 {
		return helpTitleStyle.Render(title) + "\n\n" + renderHelpSections(keys.sections)
	}

	boxW := helpBoxWidth(width)

	if vp != nil {
		content := vp.View()

		// Scroll indicator when content overflows.
		if vp.TotalLineCount() > vp.VisibleLineCount() {
			pct := int(vp.ScrollPercent() * 100)
			arrows := ""
			if vp.ScrollPercent() > 0 {
				arrows += "▲ "
			}
			if vp.ScrollPercent() < 1 {
				arrows += "▼ "
			}
			content += "\n" + dim.Render(fmt.Sprintf("%s%d%%", arrows, pct))
		}

		box := helpBoxStyle.Width(boxW).Render(content)
		return withThemeBG(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box))
	}

	// Non-viewport fallback (legacy).
	content := helpContent(title, keys)
	box := helpBoxStyle.Width(boxW).Render(content)
	return withThemeBG(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box))
}
