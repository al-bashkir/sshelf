package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// highlightMatches renders matched character positions in accent style.
// positions contains the byte offsets of the runes that matched a fuzzy
// search query, as fuzzy.Match.MatchedIndexes gives them; an offset that
// does not start a rune is ignored.
// The result is the same string with matched chars styled.
func highlightMatches(s string, positions []int, active bool) string {
	style := checkedStyle // accent bold
	if active {
		style = lipgloss.NewStyle().Underline(true)
	}

	var sb strings.Builder
	start, inMatch := 0, false
	for i := range s { // i is the byte offset of each rune
		if m := slices.Contains(positions, i); m != inMatch {
			sb.WriteString(markRun(s[start:i], inMatch, style))
			start, inMatch = i, m
		}
	}
	sb.WriteString(markRun(s[start:], inMatch, style))
	return sb.String()
}

// markRun styles a run of matched runes; other text stays as it is.
func markRun(run string, matched bool, style lipgloss.Style) string {
	if matched {
		return style.Render(run)
	}
	return run
}

// truncateTail cuts s to max cells and ends it with "…". It counts cells,
// so wide letters and emoji fit too.
func truncateTail(s string, max int) string {
	return ansi.Truncate(s, max, "…")
}

// truncateFade truncates with a soft fade: the last visible character and
// the ellipsis are rendered in dim gray instead of a hard cut. s must be
// plain text: the cut takes a byte prefix of it.
func truncateFade(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 2 {
		return dim.Render("…") + strings.Repeat(" ", max-1)
	}

	normal := ansi.Truncate(s, max-2, "")
	// The dim letter gets the cells left before "…": one, or two when a
	// wide letter did not fit in normal.
	dimChar := ansi.Truncate(s[len(normal):], max-1-lipgloss.Width(normal), "")
	return normal + dim.Render(dimChar+"…")
}

// rowCursor is the leading cursor cell. It carries no ANSI of its own so
// rowActiveStyle fills the active row uniformly.
func rowCursor(active bool) string {
	if active {
		return "▸ "
	}
	return "  "
}

// composeRow assembles one list row: prefix + text + suffix, the suffix
// right after the text. text is truncated to the remaining width (hard cut
// when active, faded otherwise), fuzzy matches are highlighted, and the
// active row is padded to the full width so its highlight bar is solid.
// suffixW is the *styled* width of suffix, so columns do not shift as the
// cursor moves.
func composeRow(width int, active bool, prefix, text, suffix string, suffixW int, matched []int, dimText bool) string {
	text, suffix = fitRowText(width, active, prefix, text, suffix, suffixW)
	// Marks stop where the fade's dim style starts: an offset past it would
	// land inside its escape code.
	head, _, _ := strings.Cut(text, "\x1b")
	text = highlightMatches(head, matched, active) + text[len(head):]
	if dimText {
		text = dim.Render(text)
	}

	line := prefix + text + suffix
	if !active {
		return line
	}
	line += strings.Repeat(" ", max(0, width-lipgloss.Width(line)))
	// A reset inside the row (after a marked letter) also ends the bar, so
	// start the bar again after each one.
	bar, _, _ := strings.Cut(rowActiveStyle.Render("x"), "x")
	return rowActiveStyle.Render(strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bar))
}

// fitRowText cuts text to the cells that prefix and suffix leave in width.
// When even an empty text leaves no room for the suffix, it drops the
// suffix. A width <= 0 cuts nothing.
func fitRowText(width int, active bool, prefix, text, suffix string, suffixW int) (string, string) {
	if width <= 0 {
		return text, suffix
	}
	avail := width - lipgloss.Width(prefix) - suffixW
	if avail < 0 {
		// Not enough room for the badge — drop it.
		suffix = ""
		avail = max(0, width-lipgloss.Width(prefix))
	}
	if active {
		return truncateTail(text, avail), suffix
	}
	return truncateFade(text, avail), suffix
}

func renderHostLikeRow(width int, active bool, selected bool, host string, hasCfg bool, hidden bool, matchedIndexes []int) string {
	checked := "◻"
	if selected {
		checked = "◼"
	}
	if !active {
		// Style only on inactive rows; active rows get uniform rowActiveStyle.
		if selected {
			checked = checkedStyle.Render(checked)
		} else {
			checked = uncheckedStyle.Render(checked)
		}
	}
	prefix := rowCursor(active) + checked + " "

	// Always reserve the badge's styled width, active or not, so the host
	// name column does not shift when the cursor moves.
	suffix, suffixW := "", 0
	if hasCfg {
		styledCfg := " " + badgeCfgStyle.Render("⚙")
		suffixW = lipgloss.Width(styledCfg)
		suffix = styledCfg
		if active {
			// Same visual width as the styled badge (padding 0,1).
			suffix = "  ⚙ "
		}
	}

	// Hidden hosts get a ⊘ marker. That shifts the display string, so their
	// match indexes (relative to the raw name) no longer apply.
	if hidden {
		return composeRow(width, active, prefix, "⊘ "+host, suffix, suffixW, nil, !active)
	}
	return composeRow(width, active, prefix, host, suffix, suffixW, matchedIndexes, false)
}

func renderSimpleRow(width int, active bool, text string, matchedIndexes []int) string {
	return composeRow(width, active, rowCursor(active), text, "", 0, matchedIndexes, false)
}

// renderGroupRow indents a group two cells per depth level.
func renderGroupRow(width int, active bool, name string, depth, hostCount int, matchedIndexes []int) string {
	countStr := fmt.Sprintf("%d", hostCount)
	styledBadge := " " + badgeCountStyle.Render(countStr)
	suffix := styledBadge
	if active {
		// Plain text occupying the same width as the styled badge
		// (badgeCountStyle has Padding(0,1) = 1 space each side).
		suffix = "  " + countStr + " "
	}
	return composeRow(width, active, rowCursor(active)+strings.Repeat("  ", depth), name, suffix, lipgloss.Width(styledBadge), matchedIndexes, false)
}
