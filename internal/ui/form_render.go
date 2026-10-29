package ui

import (
	"strings"
)

// ---------------------------------------------------------------------------
// Content rendering
// ---------------------------------------------------------------------------

// renderFormContent builds all visible lines for the form content area.
// Returns the lines and the line index of the focused item (for scrolling).
func (m *formModel) renderFormContent(innerW int) ([]string, int) {
	lines := make([]string, 0, 40)
	focusLine := 0
	prevSectionIdx := -1
	fieldW := max(10, innerW-m.fieldIndent-m.labelWidth-1)

	for i, item := range m.items {
		focused := i == m.focusIdx
		sec := &m.schema.Sections[item.sectionIdx]

		// Render a section separator before the first field of each section.
		if item.sectionIdx != prevSectionIdx {
			lines = append(lines, formSection(sec.Label, innerW))
			prevSectionIdx = item.sectionIdx
		}

		fd := &sec.Fields[item.fieldIdx]
		if focused {
			focusLine = len(lines)
		}
		lines = append(lines, m.renderFieldLines(fd, focused, fieldW)...)
	}

	return lines, focusLine
}

// ---------------------------------------------------------------------------
// Field rendering
// ---------------------------------------------------------------------------

// renderFieldLines renders a single field as one or more lines.
// The focused field gets a helper text line appended.
func (m *formModel) renderFieldLines(fd *fieldDef, focused bool, fieldW int) []string {
	var lines []string

	// Check disabled state.
	disabledReason := m.isFieldDisabled(fd)
	disabled := disabledReason != ""

	// Disabled fields cannot be focused (navigation skips them),
	// but guard defensively.
	if disabled {
		focused = false
	}

	dirty := m.originals[fd.Key] != m.values[fd.Key]

	indent := strings.Repeat(" ", m.fieldIndent)
	prefix := indent
	if focused {
		// Replace indent with focus indicator.
		if m.fieldIndent >= 2 {
			prefix = "> "
			if m.fieldIndent > 2 {
				prefix = strings.Repeat(" ", m.fieldIndent-2) + "> "
			}
		} else {
			prefix = ">"
		}
	} else if dirty && m.fieldIndent >= 1 {
		// Show modified indicator on unfocused dirty fields.
		prefix = statusWarn.Render("•") + strings.Repeat(" ", m.fieldIndent-1)
	}

	// Disabled fields: render label + value entirely in disabledStyle.
	if disabled {
		labelStr := disabledStyle.Render(padVisible(fd.Label+":", m.labelWidth))
		val := strings.TrimSpace(m.values[fd.Key])
		if val == "" {
			val = "default"
		}
		lines = append(lines, prefix+labelStr+" "+disabledStyle.Render(val))
		// Show disable reason below.
		helperPrefix := strings.Repeat(" ", m.fieldIndent)
		lines = append(lines, disabledStyle.Render(helperPrefix+"╰ "+disabledReason))
		return lines
	}

	labelStr := formLabel(fd.Label+":", m.labelWidth, focused)

	switch fd.Kind {
	case fieldText, fieldNumber:
		in := m.input(fd.Key)
		w := fieldW
		if fd.Narrow {
			w = min(12, fieldW)
		}
		if in != nil {
			lines = append(lines, prefix+labelStr+" "+underlineInput(*in, w))
		} else {
			lines = append(lines, prefix+labelStr+" "+underlineText(m.values[fd.Key], focused, w))
		}

	case fieldPicker:
		cur := strings.TrimSpace(m.values[fd.Key])
		lines = append(lines, prefix+labelStr+" "+renderPickerInline(fd.Options, cur, focused, fieldW))

	case fieldSubModal:
		val := strings.TrimSpace(m.values[fd.Key])
		displayVal := val
		if displayVal == "" {
			displayVal = "default"
		}
		rendered := padVisible(displayVal, fieldW)
		if focused {
			rendered = checkedStyle.Render(rendered)
		} else {
			rendered = dim.Render(rendered)
		}
		lines = append(lines, prefix+labelStr+" "+rendered)
	}

	// Validation error or helper text below focused field.
	if focused {
		helperPrefix := strings.Repeat(" ", m.fieldIndent)
		if errMsg := m.validationErrs[fd.Key]; errMsg != "" {
			lines = append(lines, statusErr.Render(helperPrefix+"╰ "+errMsg))
		} else if fd.Helper != "" && m.showFieldHelp {
			lines = append(lines, hintStyle.Render(helperPrefix+"╰ "+fd.Helper))
		}
	}

	return lines
}

// ---------------------------------------------------------------------------
// Picker inline rendering (value + ▾ indicator)
// ---------------------------------------------------------------------------

// renderPickerInline renders the current picker value with a dropdown
// indicator. The actual option list is shown as a popup overlay.
func renderPickerInline(options []fieldOption, cur string, focused bool, maxW int) string {
	display := cur
	for _, opt := range options {
		if strings.TrimSpace(opt.Value) == cur {
			display = opt.Display
			break
		}
	}
	if display == "" {
		display = "default"
	}

	indicator := dim.Render(" ▾")
	if focused {
		display = checkedStyle.Render(display)
	}

	text := display + indicator
	return padVisible(text, maxW)
}
