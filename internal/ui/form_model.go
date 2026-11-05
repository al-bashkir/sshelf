package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Navigation item
// ---------------------------------------------------------------------------

// formNavItem is a single navigable item (always a field) in the form.
type formNavItem struct {
	sectionIdx int
	fieldIdx   int
}

// ---------------------------------------------------------------------------
// Text input entry
// ---------------------------------------------------------------------------

// formTextInput pairs a field key with its textinput.Model.
type formTextInput struct {
	key   string
	input textinput.Model
}

// ---------------------------------------------------------------------------
// formModel
// ---------------------------------------------------------------------------

// formModel manages the generic state of a form with vim-style navigation
// and declarative field definitions.
//
// It is not a tea.Model itself — specific form models (defaults, group, host)
// embed it and delegate to its methods.
type formModel struct {
	schema    formSchema
	values    map[string]string // current values by field key
	originals map[string]string // snapshot at creation (for dirty check)

	items    []formNavItem // navigable field items
	focusIdx int           // index into items
	editing  bool          // true when in insert mode

	inputs []formTextInput // text inputs for text/number fields

	picker *optionPickerPopup // non-nil when a picker popup is open

	validationErrs map[string]string // per-field inline validation errors

	discardExit tea.Cmd // exit awaiting y/n while non-nil (unsaved changes)

	width  int
	height int

	labelWidth    int  // label column width (chars)
	fieldIndent   int  // indent for fields under section headers
	showFieldHelp bool // show helper text below focused fields
}

// newFormModel creates a formModel from a schema and initial values.
func newFormModel(schema formSchema, values map[string]string, labelWidth int, showFieldHelp bool) formModel {
	originals := make(map[string]string, len(values))
	for k, v := range values {
		originals[k] = v
	}

	m := formModel{
		schema:         schema,
		values:         values,
		originals:      originals,
		validationErrs: make(map[string]string),
		labelWidth:     labelWidth,
		fieldIndent:    2,
		showFieldHelp:  showFieldHelp,
	}

	// Create text inputs for text/number fields.
	for si := range schema.Sections {
		for fi := range schema.Sections[si].Fields {
			fd := &schema.Sections[si].Fields[fi]
			if !fd.isTextField() {
				continue
			}
			in := textinput.New()
			limit := fd.CharLimit
			if limit <= 0 {
				limit = 256
			}
			in.CharLimit = limit
			in.Prompt = ""
			in.SetValue(values[fd.Key])
			in.Placeholder = fd.Placeholder
			configureSearch(&in)
			m.inputs = append(m.inputs, formTextInput{key: fd.Key, input: in})
		}
	}

	m.rebuildItems()
	// Focus the first enabled field (the first field when all are disabled).
	for i := range m.items {
		if m.isFieldDisabled(m.fieldAt(i)) == "" {
			m.focusIdx = i
			break
		}
	}
	m.applyFocusStyles()
	return m
}

// ---------------------------------------------------------------------------
// Value access
// ---------------------------------------------------------------------------

// input returns a pointer to the textinput.Model for the given field key.
func (m *formModel) input(key string) *textinput.Model {
	for i := range m.inputs {
		if m.inputs[i].key == key {
			return &m.inputs[i].input
		}
	}
	return nil
}

// value returns the current form value for a field key.
func (m *formModel) value(key string) string {
	return m.values[key]
}

// setValue updates the current form value for a field key.
func (m *formModel) setValue(key, val string) {
	m.values[key] = val
}

// isDirty returns true if any value differs from the original snapshot.
func (m *formModel) isDirty() bool {
	for key, val := range m.values {
		if m.originals[key] != val {
			return true
		}
	}
	return false
}

// resetOriginals snapshots the current values as the new baseline,
// clearing the dirty state.
func (m *formModel) resetOriginals() {
	for k, v := range m.values {
		m.originals[k] = v
	}
}

// ---------------------------------------------------------------------------
// Navigation items
// ---------------------------------------------------------------------------

// rebuildItems constructs the flat navigation list of fields.
func (m *formModel) rebuildItems() {
	m.items = m.items[:0]
	for si, sec := range m.schema.Sections {
		for fi := range sec.Fields {
			m.items = append(m.items, formNavItem{sectionIdx: si, fieldIdx: fi})
		}
	}
}

// ---------------------------------------------------------------------------
// Focus queries
// ---------------------------------------------------------------------------

// fieldAt returns the field of navigation item i.
func (m *formModel) fieldAt(i int) *fieldDef {
	it := m.items[i]
	return &m.schema.Sections[it.sectionIdx].Fields[it.fieldIdx]
}

// focusedField returns the definition of the focused field, or nil in a
// form with no fields. focusIdx is always a valid item otherwise.
func (m *formModel) focusedField() *fieldDef {
	if m.focusIdx >= len(m.items) {
		return nil
	}
	return m.fieldAt(m.focusIdx)
}

// focusedFieldKey returns the key of the focused field, or "".
func (m *formModel) focusedFieldKey() string {
	if fd := m.focusedField(); fd != nil {
		return fd.Key
	}
	return ""
}

// ---------------------------------------------------------------------------
// Disabled field check
// ---------------------------------------------------------------------------

// isFieldDisabled returns the disable reason for a field, or "" if enabled.
func (m *formModel) isFieldDisabled(fd *fieldDef) string {
	if fd == nil || fd.DisabledWhen == nil {
		return ""
	}
	return fd.DisabledWhen(m.values)
}

// enabledField returns the focused field, or nil when no field has focus
// or the focused one is disabled.
func (m *formModel) enabledField() *fieldDef {
	fd := m.focusedField()
	if m.isFieldDisabled(fd) != "" {
		return nil
	}
	return fd
}

// ---------------------------------------------------------------------------
// Focus management
// ---------------------------------------------------------------------------

// applyFocusStyles updates text input accent styling to match current focus.
// Also used to re-apply styles after a theme/accent change.
func (m *formModel) applyFocusStyles() {
	focusedKey := m.focusedFieldKey()
	for i := range m.inputs {
		setSearchFocused(&m.inputs[i].input, m.inputs[i].key == focusedKey)
	}
}

// setFocusIdx moves focus to item idx.
func (m *formModel) setFocusIdx(idx int) {
	m.exitEdit()
	m.focusIdx = idx
	m.applyFocusStyles()
}

// moveFocus moves focus by delta items (+1 down, -1 up), wrapping at edges.
// Disabled fields are skipped; if all fields are disabled, focus stays put.
func (m *formModel) moveFocus(delta int) {
	n := len(m.items)
	pos := m.focusIdx
	for range n {
		pos = (pos + delta + n) % n
		if m.isFieldDisabled(m.fieldAt(pos)) == "" {
			m.setFocusIdx(pos)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Edit mode (insert)
// ---------------------------------------------------------------------------

// enterEdit activates insert mode for the focused text/number field.
func (m *formModel) enterEdit() {
	fd := m.focusedField()
	if fd == nil || !fd.isTextField() {
		return
	}
	m.editing = true
	if in := m.input(fd.Key); in != nil {
		_ = in.Focus()
	}
}

// exitEdit deactivates insert mode and syncs the text input value.
func (m *formModel) exitEdit() {
	if !m.editing {
		return
	}
	m.editing = false
	fd := m.focusedField() // insert mode is only ever on a text field
	in := m.input(fd.Key)
	m.values[fd.Key] = in.Value()
	in.Blur()
	m.validateField(fd)
}

// updateFocusedInput types a key into the focused text input (insert mode)
// and syncs the value back.
func (m *formModel) updateFocusedInput(msg tea.Msg) tea.Cmd {
	fd := m.focusedField()
	in := m.input(fd.Key)
	var cmd tea.Cmd
	*in, cmd = in.Update(msg)
	m.values[fd.Key] = in.Value()
	return cmd
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// handleKey processes a key message in the generic form.
// Returns true if the key was consumed by the form.
func (m *formModel) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case m.picker != nil:
		m.handlePickerKey(msg)
		return true, nil
	case m.editing:
		return true, m.editKey(msg)
	}
	return m.navKey(msg.String()), nil
}

// editKey handles insert mode: Esc ends it, Enter ends it and moves on, and
// every other key types into the field.
func (m *formModel) editKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.exitEdit()
	case "enter":
		m.exitEdit()
		m.moveFocus(1)
	default:
		return m.updateFocusedInput(msg)
	}
	return nil
}

// navKey handles normal mode. It returns false for keys the wrapper owns.
func (m *formModel) navKey(s string) bool {
	switch s {
	case "j", "down", "tab":
		m.moveFocus(1)
	case "k", "up", "shift+tab":
		m.moveFocus(-1)
	case "i":
		if m.enabledField() != nil {
			m.enterEdit()
		}
	case "enter", " ":
		return m.choose()
	case "l", "right":
		return m.open(m.enabledField())
	default:
		return false
	}
	return true
}

// choose is Enter or Space: it edits a text field and opens other fields.
// With no enabled field, focus moves on.
func (m *formModel) choose() bool {
	fd := m.enabledField()
	switch {
	case fd == nil:
		m.moveFocus(1)
	case fd.isTextField():
		m.enterEdit()
	default:
		return m.open(fd)
	}
	return true
}

// open opens a picker and leaves a sub-modal to the wrapper (false).
func (m *formModel) open(fd *fieldDef) bool {
	if fd != nil && fd.Kind == fieldSubModal {
		return false
	}
	m.openPicker(fd)
	return true
}

// ---------------------------------------------------------------------------
// Picker popup
// ---------------------------------------------------------------------------

// openPicker shows the option picker popup for fd when it is a picker with
// options.
func (m *formModel) openPicker(fd *fieldDef) {
	if fd == nil || fd.Kind != fieldPicker || len(fd.Options) == 0 {
		return
	}
	m.picker = newOptionPickerPopup(fd.Key, fd.Label, m.values[fd.Key], fd.Options)
}

// handlePickerKey routes key events to the open picker popup, which takes
// every key while it is open.
func (m *formModel) handlePickerKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		m.picker.moveDown()
	case "k", "up":
		m.picker.moveUp()
	case "enter":
		m.values[m.picker.fieldKey] = m.picker.selected()
		m.validateField(m.schema.fieldByKey(m.picker.fieldKey))
		m.picker = nil
	case "esc":
		m.picker = nil
	}
}

// overlayPickerOnVisible renders the picker popup on top of visible content
// lines, positioned as a dropdown near the focused field.
// When there is not enough room below the field the popup flips above it.
// The overlay is transparent: background content to the left and right of the
// popup box remains visible.
// focusRow is the row index of the focused field within the visible slice.
func (m *formModel) overlayPickerOnVisible(visible []string, focusRow, innerW int) []string {
	if m.picker == nil {
		return visible
	}

	// Limit popup width to the value area.
	popupMaxW := innerW - m.fieldIndent - m.labelWidth
	if popupMaxW < 20 {
		popupMaxW = min(innerW, 20)
	}
	popup := m.picker.View(popupMaxW)
	popupLines := strings.Split(strings.TrimRight(popup, "\n"), "\n")

	// Decide direction: prefer below, flip above when not enough room.
	roomBelow := len(visible) - focusRow - 1
	roomAbove := focusRow

	startRow := focusRow + 1 // below by default
	if len(popupLines) > roomBelow && roomAbove > roomBelow {
		// Flip above: place popup so its last line is the row above focusRow.
		startRow = focusRow - len(popupLines)
	}

	startCol := m.fieldIndent + m.labelWidth + 1

	result := make([]string, len(visible))
	copy(result, visible)

	for i, pLine := range popupLines {
		row := startRow + i
		if row < 0 || row >= len(result) {
			continue
		}
		bg := result[row]
		popupW := lipgloss.Width(pLine)

		// Left: ANSI-aware truncation to startCol columns.
		left := ansi.Truncate(bg, startCol, "")
		// Pad if background line is shorter than startCol.
		if lw := lipgloss.Width(left); lw < startCol {
			left += strings.Repeat(" ", startCol-lw)
		}

		// Right: everything past startCol + popupW.
		right := ansi.TruncateLeft(bg, startCol+popupW, "")

		result[row] = left + pLine + right
	}

	return result
}

// ---------------------------------------------------------------------------
// Resize
// ---------------------------------------------------------------------------

// handleResize updates form dimensions and text input widths.
func (m *formModel) handleResize(w, h int) {
	m.width = w
	m.height = h
	innerW := max(0, w-2)
	fieldW := max(10, innerW-m.fieldIndent-m.labelWidth-1)
	for i := range m.inputs {
		if m.schema.fieldByKey(m.inputs[i].key).Narrow {
			m.inputs[i].input.Width = min(12, fieldW)
		} else {
			m.inputs[i].input.Width = fieldW
		}
	}
}

// ---------------------------------------------------------------------------
// Footer
// ---------------------------------------------------------------------------

// positionInfo returns a human-readable position string for the footer.
func (m *formModel) positionInfo() string {
	if m.focusIdx < 0 || m.focusIdx >= len(m.items) {
		return ""
	}
	item := m.items[m.focusIdx]
	sec := &m.schema.Sections[item.sectionIdx]
	return fmt.Sprintf("%d/%d %s", item.fieldIdx+1, len(sec.Fields), sec.Label)
}

// footerHints returns context-sensitive key hints for the footer.
// Format: "key action  key action" (double-space separated, for styledFooter).
// renderFooter shows the insert-mode hints itself.
func (m *formModel) footerHints() string {
	if m.picker != nil {
		return "⏎ select  Esc cancel"
	}
	fd := m.focusedField()
	if fd == nil {
		return "Ctrl+S save  j/k nav  Esc back"
	}
	switch fd.Kind {
	case fieldText, fieldNumber:
		return "Ctrl+S save  j/k nav  i edit  Esc back"
	case fieldPicker:
		return "Ctrl+S save  j/k nav  ⏎ choose  Esc back"
	case fieldSubModal:
		return "Ctrl+S save  j/k nav  ⏎ select  Esc back"
	}
	return "Ctrl+S save  j/k nav  Esc back"
}

// renderFooter returns the styled footer line.
func (m *formModel) renderFooter() string {
	pos := m.positionInfo()
	if m.editing {
		return footerStyle.Render(pos) + "  " + headerStyle.Render("INSERT") + "  " + styledFooter("Ctrl+S save  Esc done")
	}
	return footerStyle.Render(pos) + "  " + styledFooter(m.footerHints())
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// validateField runs the validator for a single field and updates
// validationErrs.
func (m *formModel) validateField(fd *fieldDef) {
	delete(m.validationErrs, fd.Key)
	if fd.Validate == nil {
		return
	}
	if err := fd.Validate(m.values[fd.Key]); err != nil {
		m.validationErrs[fd.Key] = err.Error()
	}
}

// ---------------------------------------------------------------------------
// Unsaved-changes guard
// ---------------------------------------------------------------------------

// exitOrAsk returns exit at once for a clean form. For a dirty form it starts
// the discard confirm, shows it in t (the owning screen's toast), keeps exit
// for a "y" answer, and returns nil.
func (m *formModel) exitOrAsk(exit tea.Cmd, t *toast) tea.Cmd {
	if !m.isDirty() {
		return exit
	}
	m.discardExit = exit
	*t = discardToast
	return nil
}

// askingDiscard reports whether the discard confirm is open.
func (m *formModel) askingDiscard() bool { return m.discardExit != nil }

// handleDiscardKey answers the open discard confirm and clears its toast t:
// y runs the held exit, n or Esc keeps editing, any other key is ignored.
func (m *formModel) handleDiscardKey(msg tea.KeyMsg, t *toast) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		exit := m.discardExit
		m.discardExit, *t = nil, toast{}
		return exit
	case "n", "N", "esc":
		m.discardExit, *t = nil, toast{}
	}
	return nil
}

// discardToast is shown while the discard confirm is open.
var discardToast = toast{text: "unsaved changes — discard? (y/n)", level: toastWarn}
