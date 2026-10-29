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

// selected returns the value of the currently highlighted option.
func (p *optionPickerPopup) selected() string {
	if p.cursor >= 0 && p.cursor < len(p.options) {
		return p.options[p.cursor].Value
	}
	return ""
}

// View renders the popup as a box-drawing framed list.
// maxW is the maximum width available (from the parent form).
func (p *optionPickerPopup) View(maxW int) string {
	if len(p.options) == 0 {
		return ""
	}

	// Measure the widest option display text.
	widest := lipgloss.Width(p.label)
	for _, opt := range p.options {
		if w := lipgloss.Width(opt.Display); w > widest {
			widest = w
		}
	}

	// Box inner width: widest option + cursor prefix ("▸ " = 2) + right pad.
	innerW := widest + 4 // "▸ " + text + " "
	if innerW < 20 {
		innerW = 20
	}
	totalW := innerW + 2 // box borders
	if maxW > 0 && totalW > maxW {
		totalW = maxW
		innerW = totalW - 2
	}

	// Build lines.
	lines := make([]string, 0, len(p.options)+4)
	lines = append(lines, focusedBoxTitleTop(totalW, p.label))

	for i, opt := range p.options {
		prefix := "  "
		if i == p.cursor {
			prefix = "▸ "
		}
		text := prefix + opt.Display
		if i == p.cursor {
			// Fill to width for highlight bar.
			pad := innerW - lipgloss.Width(text)
			if pad > 0 {
				text += strings.Repeat(" ", pad)
			}
			text = rowActiveStyle.Render(text)
		} else {
			text = padVisible(text, innerW)
		}
		lines = append(lines, focusedBoxLine(totalW, text))
	}

	lines = append(lines, focusedBoxBottom(totalW))

	return strings.Join(lines, "\n")
}
