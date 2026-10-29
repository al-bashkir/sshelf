package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// pickerStatusLine renders a modal picker's footer: caller-supplied counts,
// then page indicator, toast, and the current search query.
func pickerStatusLine(left string, l *list.Model, query string, t toast) string {
	if l.Paginator.TotalPages > 1 {
		left += "  " + dim.Render(fmt.Sprintf("pg:%d/%d", l.Paginator.Page+1, l.Paginator.TotalPages))
	}
	if !t.empty() {
		left += "  " + renderToast(t)
	}

	searchInfo := "search"
	if q := strings.TrimSpace(query); q != "" {
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		searchInfo = "search: " + q
	}
	return left + "  " + statusOK.Render(searchInfo)
}

// renderCmdPromptModal renders the "connect with remote command" modal overlay.
// parentCrumb is the breadcrumb prefix (e.g. "Hosts", "Groups > web").
// description is the instructional text shown above the input.
func renderCmdPromptModal(totalW, totalH int, parentCrumb, description string, cmdInput textinput.Model) string {
	mw, mh := modalSize(totalW, totalH, cmdPromptMaxW, cmdPromptMaxH, cmdPromptMarginW, cmdPromptMarginH)
	var b strings.Builder
	b.WriteString(description)
	b.WriteString("\n\n")
	b.WriteString(cmdInput.View())
	b.WriteString("\n")
	b.WriteString(styledFooter("⏎ connect  Esc cancel"))
	box := renderFocusedFrame(mw, mh, breadcrumbTitle(parentCrumb, "Command"), "", strings.TrimRight(b.String(), "\n"), "")
	return placeCentered(totalW, totalH, box)
}

// renderListEmptyState renders a centered empty-state message for a list view.
// When query is non-empty, shows a "no matches" message. Otherwise shows the
// caller-provided defaultMsg (which may include pre-styled hints).
func renderListEmptyState(width, height int, query, defaultMsg string) string {
	innerW := max(0, width-2)
	innerH := max(0, height-2)
	// Approximate content area: header lines + footer estimate (~2 lines).
	contentH := max(0, innerH-tabBoxHeaderLines-2)

	dots := dim.Render("·  ·  ·")
	var msg string
	q := strings.TrimSpace(query)
	if q != "" {
		msg = dots + "\n\n" + dim.Render(fmt.Sprintf("No matches for %q", q)) + "\n" + dim.Render("Esc to clear search")
	} else {
		msg = dots + "\n\n" + defaultMsg
	}

	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, msg)
}
