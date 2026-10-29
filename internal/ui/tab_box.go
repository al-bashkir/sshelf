package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var mainTabs = []string{"Hosts", "Groups", "Settings"}

// boxRule renders a horizontal box edge of total width w with the given
// corner/junction runes.
func boxRule(st lipgloss.Style, w int, left, right string) string {
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return st.Render(left + right)
	}
	return st.Render(left + strings.Repeat("─", w-2) + right)
}

// boxLineStyled renders one box row: borders plus content padded to width.
func boxLineStyled(st lipgloss.Style, w int, content string) string {
	if w <= 1 {
		return ""
	}
	border := st.Render("│")
	if w == 2 {
		return border + border
	}
	innerW := w - 2
	content = strings.TrimRight(content, "\n")
	cw := lipgloss.Width(content)
	if cw > innerW {
		content = lipgloss.NewStyle().MaxWidth(innerW).Render(content)
		cw = lipgloss.Width(content)
	}
	pad := max(0, innerW-cw)
	padding := tabBoxPadStyle.Render(strings.Repeat(" ", pad))
	// Apply theme background so gaps between styled elements inherit it.
	return border + withThemeBG(content+padding) + border
}

func boxTop(w int) string                  { return boxRule(tabBoxBorderStyle, w, "┌", "┐") }
func boxSep(w int) string                  { return boxRule(tabBoxBorderStyle, w, "├", "┤") }
func boxBottom(w int) string               { return boxRule(tabBoxBorderStyle, w, "└", "┘") }
func boxLine(w int, content string) string { return boxLineStyled(tabBoxBorderStyle, w, content) }

// Focused variants — for modals/popups using cFocusedBorder.
func focusedBoxTop(w int) string    { return boxRule(focusedBoxBorderStyle, w, "┌", "┐") }
func focusedBoxSep(w int) string    { return boxRule(focusedBoxBorderStyle, w, "├", "┤") }
func focusedBoxBottom(w int) string { return boxRule(focusedBoxBorderStyle, w, "└", "┘") }
func focusedBoxLine(w int, content string) string {
	return boxLineStyled(focusedBoxBorderStyle, w, content)
}

// focusedBoxTitleTop renders a top border with the title embedded in it.
func focusedBoxTitleTop(w int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" || w <= 2 {
		return focusedBoxTop(w)
	}
	innerW := w - 2
	seg := " " + title + " "
	if lipgloss.Width(seg) > innerW {
		seg = " " + truncateTail(title, max(0, innerW-2)) + " "
	}
	fill := max(0, innerW-lipgloss.Width(seg))
	// Style the non-title border parts; keep title unstyled so its own colors show.
	return focusedBoxBorderStyle.Render("┌") + seg + focusedBoxBorderStyle.Render(strings.Repeat("─", fill)+"┐")
}

func padVisible(s string, width int) string {
	if width <= 0 {
		return s
	}
	s = strings.TrimRight(s, "\n")
	cw := lipgloss.Width(s)
	if cw > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
		cw = lipgloss.Width(s)
	}
	if pad := width - cw; pad > 0 {
		s += tabBoxPadStyle.Render(strings.Repeat(" ", pad))
	}
	return s
}

func renderTabsLine(active int, tabs []string) string {
	parts := make([]string, 0, len(tabs))
	for i, t := range tabs {
		if i == active {
			parts = append(parts, tabActiveStyle.Render(t))
		} else {
			parts = append(parts, tabInactiveStyle.Render(t))
		}
	}
	return strings.Join(parts, "  ")
}

func renderMainTabBox(width, height int, activeTab int, headerLeft string, headerRight string, listView string) string {
	return renderMainTabBoxWithFooter(width, height, activeTab, headerLeft, headerRight, listView, "")
}

func renderMainTabBoxWithFooter(width, height int, activeTab int, headerLeft string, headerRight string, listView string, footer string) string {
	return renderTabBox(width, height, renderTabsLine(activeTab, mainTabs), headerLeft, headerRight, listView, footer)
}

func renderBreadcrumbTabBox(width, height int, breadcrumb string, headerLeft string, headerRight string, listView string, footer string) string {
	return renderTabBox(width, height, breadcrumb, headerLeft, headerRight, listView, footer)
}

// renderTabBox draws the full-screen box: a top line (tabs or breadcrumb),
// a header row, the content area padded to fill, and an optional footer.
func renderTabBox(width, height int, topLine string, headerLeft string, headerRight string, listView string, footer string) string {
	if width <= 0 || height <= 0 {
		// Fallback before we know the window size.
		return strings.TrimRight(topLine+"\n"+headerLeft+"\n"+listView, "\n")
	}
	if height < 3 {
		return boxTop(width)
	}

	innerW := max(0, width-2)
	innerH := max(0, height-2)

	fixed := tabBoxHeaderLines // top line + sep + header + sep
	hasFooter := strings.TrimSpace(footer) != ""
	var footerLines []string
	if hasFooter {
		footerLines = strings.Split(strings.TrimRight(footer, "\n"), "\n")
		fixed += 1 + len(footerLines) // sep + N footer lines
	}
	contentH := max(0, innerH-fixed)

	var contentLines []string
	if content := strings.TrimRight(listView, "\n"); strings.TrimSpace(content) != "" {
		contentLines = strings.Split(content, "\n")
	}

	out := make([]string, 0, height)
	out = append(out, boxTop(width))
	out = append(out, boxLine(width, padVisible(topLine, innerW)))
	out = append(out, boxSep(width))
	out = append(out, boxLine(width, padVisible(joinHeader(innerW, headerLeft, headerRight), innerW)))
	out = append(out, boxSep(width))

	for i := 0; i < contentH; i++ {
		line := tabBoxPadStyle.Render(strings.Repeat(" ", innerW))
		if i < len(contentLines) {
			line = padVisible(contentLines[i], innerW)
		}
		out = append(out, boxLine(width, line))
	}
	if hasFooter {
		out = append(out, boxSep(width))
		for _, fl := range footerLines {
			out = append(out, boxLine(width, padVisible(fl, innerW)))
		}
	}
	out = append(out, boxBottom(width))
	return strings.Join(out, "\n")
}
