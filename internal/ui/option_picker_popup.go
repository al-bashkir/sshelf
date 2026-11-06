package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// optionPickerPopup is a lightweight popup for selecting one option from a
// fieldDef.Options list.  It is managed by formModel and rendered as an
// overlay on top of the form content.
type optionPickerPopup struct {
	fieldKey string
	label    string
	options  []fieldOption
	cursor   int // index of the highlighted option
}

// newOptionPickerPopup creates a popup pre-positioned on the current value.
func newOptionPickerPopup(fieldKey, label, currentValue string, options []fieldOption) *optionPickerPopup {
	cur := 0
	cv := strings.TrimSpace(currentValue)
	for i, opt := range options {
		if strings.TrimSpace(opt.Value) == cv {
			cur = i
			break
		}
	}
	return &optionPickerPopup{
		fieldKey: fieldKey,
		label:    label,
		options:  options,
		cursor:   cur,
	}
}

// moveUp moves the cursor up, wrapping at the top.
func (p *optionPickerPopup) moveUp() {
	p.cursor--
	if p.cursor < 0 {
		p.cursor = len(p.options) - 1
	}
}

// moveDown moves the cursor down, wrapping at the bottom.
func (p *optionPickerPopup) moveDown() {
	p.cursor++
	if p.cursor >= len(p.options) {
		p.cursor = 0
	}
}

// selected returns the value of the highlighted option; the cursor always
// sits on one, as the popup opens only with options.
func (p *optionPickerPopup) selected() string {
	return p.options[p.cursor].Value
}

// View renders the popup as a box-drawing framed list.
// maxW is the maximum width available (from the parent form). The box shows
// at most maxRows options, a window centered on the cursor where the ends of
// the list allow.
// formModel.openPicker opens it only for a field with options.
func (p *optionPickerPopup) View(maxW, maxRows int) string {
	// Measure the widest option display text.
	widest := lipgloss.Width(p.label)
	for _, opt := range p.options {
		widest = max(widest, lipgloss.Width(opt.Display))
	}

	// Box inner width: widest option + cursor prefix ("▸ " = 2) + right pad.
	innerW := max(20, widest+4) // "▸ " + text + " "
	totalW := innerW + 2        // box borders
	if maxW > 0 && totalW > maxW {
		totalW = maxW
		innerW = totalW - 2
	}

	// Build lines.
	lines := make([]string, 0, len(p.options)+4)
	lines = append(lines, focusedBoxTitleTop(totalW, p.label))

	first := max(0, min(p.cursor-maxRows/2, len(p.options)-maxRows))
	for i, opt := range p.options[first:min(len(p.options), first+maxRows)] {
		text := "  " + opt.Display
		if first+i == p.cursor {
			// Fill to width for highlight bar.
			text = "▸ " + opt.Display
			text = rowActiveStyle.Render(text + strings.Repeat(" ", max(0, innerW-lipgloss.Width(text))))
		} else {
			text = padVisible(text, innerW)
		}
		lines = append(lines, focusedBoxLine(totalW, text))
	}

	lines = append(lines, focusedBoxBottom(totalW))

	return strings.Join(lines, "\n")
}
