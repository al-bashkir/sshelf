package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// highlightMatches renders matched character positions in accent style.
// positions contains rune indexes that matched a fuzzy search query.
// The result is the same string with matched chars styled.
func highlightMatches(s string, positions []int, active bool) string {
	if len(positions) == 0 {
		return s
	}

	matchSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		matchSet[p] = true
	}

	runes := []rune(s)
	var sb strings.Builder
	sb.Grow(len(s) * 2) // rough estimate

	style := checkedStyle // accent bold
	if active {
		style = lipgloss.NewStyle().Underline(true)
	}

	run := false
	runStart := 0
	for i := 0; i <= len(runes); i++ {
		inMatch := i < len(runes) && matchSet[i]
		if inMatch && !run {
			// Flush normal text.
			if i > runStart {
				sb.WriteString(string(runes[runStart:i]))
			}
			run = true
			runStart = i
		} else if !inMatch && run {
			// Flush matched text.
			sb.WriteString(style.Render(string(runes[runStart:i])))
			run = false
			runStart = i
		}
	}
	// Flush remaining.
	if run {
		sb.WriteString(style.Render(string(runes[runStart:])))
	} else if runStart < len(runes) {
		sb.WriteString(string(runes[runStart:]))
	}

	return sb.String()
}

func truncateTail(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}

	r := []rune(s)
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max-1]) + "…"
}

// truncateFade truncates with a soft fade: the last visible character and
// the ellipsis are rendered in dim gray instead of a hard cut.
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

	r := []rune(s)
	cutoff := max - 2
	if cutoff > len(r) {
		cutoff = len(r)
	}
	normal := string(r[:cutoff])
	dimChar := ""
	if cutoff < len(r) {
		dimChar = string(r[cutoff : cutoff+1])
	}
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

// composeRow assembles one list row: prefix + text + right-aligned suffix.
// text is truncated to the remaining width (hard cut when active, faded
// otherwise), fuzzy matches are highlighted, and the active row is padded to
// the full width so its highlight bar is solid. suffixW is the *styled* width
// of suffix, so columns do not shift as the cursor moves.
func composeRow(width int, active bool, prefix, text, suffix string, suffixW int, matched []int, dimText bool) string {
	if width > 0 {
		avail := width - lipgloss.Width(prefix) - suffixW
		if avail < 0 {
			// Not enough room for the badge — drop it.
			suffix = ""
			avail = max(0, width-lipgloss.Width(prefix))
		}
		if active {
			text = truncateTail(text, avail)
		} else {
			text = truncateFade(text, avail)
		}
	}

	if len(matched) > 0 {
		visibleLen := len([]rune(text))
		shown := make([]int, 0, len(matched))
		for _, idx := range matched {
			if idx >= 0 && idx < visibleLen {
				shown = append(shown, idx)
			}
		}
		if len(shown) > 0 {
			text = highlightMatches(text, shown, active)
		}
	}
	if dimText {
		text = dim.Render(text)
	}

	line := prefix + text + suffix
	if active {
		if width > 0 {
			if need := width - lipgloss.Width(line); need > 0 {
				line += strings.Repeat(" ", need)
			}
		}
		return rowActiveStyle.Render(line)
	}
	return line
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

func renderGroupRow(width int, active bool, name string, hostCount int, matchedIndexes []int) string {
	countStr := fmt.Sprintf("%d", hostCount)
	styledBadge := " " + badgeCountStyle.Render(countStr)
	suffix := styledBadge
	if active {
		// Plain text occupying the same width as the styled badge
		// (badgeCountStyle has Padding(0,1) = 1 space each side).
		suffix = "  " + countStr + " "
	}
	return composeRow(width, active, rowCursor(active), name, suffix, lipgloss.Width(styledBadge), matchedIndexes, false)
}
