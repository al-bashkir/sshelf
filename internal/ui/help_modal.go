package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
		boxW = min(termW-2, 30) // the border takes 2 more
	}
	return boxW
}

// helpInnerWidth is the viewport width: the box less border (2) and
// padding (2*2). At width 0 the viewport would not wrap or cut its lines,
// and the box would grow taller than the window.
func helpInnerWidth(boxW int) int {
	return max(1, boxW-confirmDialogPadding)
}

// initHelpViewport creates a viewport sized for the help modal and sets its content.
func initHelpViewport(width, height int, title string, keys helpMap) viewport.Model {
	boxW := helpBoxWidth(width)
	innerW := helpInnerWidth(boxW)

	content := helpContent(title, keys)

	// Show every line when they fit; else scroll, keeping a row for the
	// scroll indicator.
	vpH := strings.Count(content, "\n") + 1
	if room := height - helpBoxOverhead; vpH > room {
		vpH = max(3, room-1)
	}

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

// toggleHelpView opens or closes a screen's help overlay. Help is laid out
// only once the window size is known.
func toggleHelpView(show *bool, vp *viewport.Model, w, h int, title string, keys func() helpMap) {
	*show = !*show
	if *show && w > 0 && h > 0 {
		*vp = initHelpViewport(w, h, title, keys())
	}
}

// helpViewKey handles a key while help shows: the help key or Esc closes
// it, any other key scrolls it.
func helpViewKey(msg tea.KeyMsg, helpKey key.Binding, show *bool, vp *viewport.Model) {
	if key.Matches(msg, helpKey) || msg.String() == "esc" {
		*show = false
		return
	}
	updateHelpViewport(vp, msg)
}
