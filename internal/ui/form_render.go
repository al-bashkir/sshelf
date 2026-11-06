package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// renderFieldLines renders a single field as one or more lines: the field
// row, then the reason it is disabled, or under the focused field its
// error or help text.
func (m *formModel) renderFieldLines(fd *fieldDef, focused bool, fieldW int) []string {
	if reason := m.isFieldDisabled(fd); reason != "" {
		return m.disabledFieldLines(fd, reason)
	}
	lines := []string{m.fieldPrefix(fd, focused) + formLabel(fd.Label+":", m.labelWidth, focused) + " " + m.fieldValue(fd, focused, fieldW)}
	if note := m.fieldNote(fd, focused); note != "" {
		lines = append(lines, note)
	}
	return lines
}

// fieldPrefix is the indent before a field's label: "> " ends it on the
// focused field, "•" starts it on a changed one. newFormModel sets the
// indent to 2, so both marks fit.
func (m *formModel) fieldPrefix(fd *fieldDef, focused bool) string {
	switch {
	case focused:
		return strings.Repeat(" ", m.fieldIndent-2) + "> "
	case m.originals[fd.Key] != m.values[fd.Key]:
		return statusWarn.Render("•") + strings.Repeat(" ", m.fieldIndent-1)
	}
	return strings.Repeat(" ", m.fieldIndent)
}

// disabledFieldLines renders a disabled field and the reason below it, all
// in disabledStyle. Navigation skips disabled fields, so one that turns
// disabled under the focus draws as unfocused.
func (m *formModel) disabledFieldLines(fd *fieldDef, reason string) []string {
	label := disabledStyle.Render(padVisible(fd.Label+":", m.labelWidth))
	return []string{
		m.fieldPrefix(fd, false) + label + " " + disabledStyle.Render(valueOrDefault(m.values[fd.Key])),
		m.noteLine(disabledStyle, reason),
	}
}

// fieldValue renders the value column of an enabled field.
func (m *formModel) fieldValue(fd *fieldDef, focused bool, fieldW int) string {
	switch fd.Kind {
	case fieldPicker:
		return renderPickerInline(fd.Options, strings.TrimSpace(m.values[fd.Key]), focused, fieldW)
	case fieldSubModal:
		style := dim
		if focused {
			style = checkedStyle
		}
		return style.Render(padVisible(valueOrDefault(m.values[fd.Key]), fieldW))
	}
	if fd.Narrow {
		fieldW = min(12, fieldW)
	}
	// newFormModel makes an input for every text and number field.
	return underlineInput(*m.input(fd.Key), fieldW)
}

// fieldNote is the line under the focused field: its validation error,
// else its help text when field help is on; "" when there is none.
func (m *formModel) fieldNote(fd *fieldDef, focused bool) string {
	switch {
	case !focused:
		return ""
	case m.validationErrs[fd.Key] != "":
		return m.noteLine(statusErr, m.validationErrs[fd.Key])
	case fd.Helper != "" && m.showFieldHelp:
		return m.noteLine(hintStyle, fd.Helper)
	}
	return ""
}

// noteLine renders text as a "╰ " line under a field.
func (m *formModel) noteLine(style lipgloss.Style, text string) string {
	return style.Render(strings.Repeat(" ", m.fieldIndent) + "╰ " + text)
}

// valueOrDefault trims a field value and shows an empty one as "default".
func valueOrDefault(v string) string {
	if v = strings.TrimSpace(v); v == "" {
		return "default"
	}
	return v
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
