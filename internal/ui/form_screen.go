package ui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// formScreen is the state and key flow the host, group and settings forms
// share. Each screen embeds it and passes its own formHooks to update.
type formScreen struct {
	form          formModel
	width, height int
	toast         toast
	keymap        keyMap
	parentCrumb   string

	borderPicker *paneBorderFormatsModel // the border_fmt sub-modal while it is open
}

// formHooks is what one form screen adds to the shared flow.
type formHooks struct {
	save    func() (tea.Msg, error)          // Ctrl+S: the save message, or why the values are bad
	cancel  tea.Msg                          // Esc, once unsaved changes are answered
	ownKey  func(tea.KeyMsg) (tea.Cmd, bool) // the screen's own keys; may be nil
	borders func() *paneBorderFormatsModel   // the border_fmt sub-modal; nil without that field
}

func newFormScreen(form formModel) formScreen {
	return formScreen{form: form, keymap: defaultKeyMap()}
}

// update handles the messages every form screen shares.
func (s *formScreen) update(msg tea.Msg, h formHooks) tea.Cmd {
	switch msg := msg.(type) {
	case paneBorderFormatsCancelMsg:
		s.borderPicker = nil
	case paneBorderFormatsDoneMsg:
		s.form.setValue("border_fmt", strings.TrimSpace(msg.value))
		s.borderPicker = nil
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		s.form.handleResize(msg.Width, msg.Height)
		s.sizeBorders()
	case tea.KeyMsg:
		return s.key(msg, h)
	}
	return nil
}

// key sends a key to the open sub-modal, the unsaved-changes question,
// insert mode or the picker, in that order, and otherwise to normalKey.
func (s *formScreen) key(msg tea.KeyMsg, h formHooks) tea.Cmd {
	switch {
	case s.borderPicker != nil:
		_, cmd := s.borderPicker.Update(msg)
		return cmd
	case s.form.askingDiscard():
		return s.form.handleDiscardKey(msg, &s.toast)
	case s.form.editing || s.form.picker != nil:
		_, cmd := s.form.handleKey(msg)
		return cmd
	}
	return s.normalKey(msg, h)
}

// normalKey handles a key out of insert mode and the picker: the screen's
// own keys, Ctrl+S, Esc, the border_fmt sub-modal, then form navigation.
func (s *formScreen) normalKey(msg tea.KeyMsg, h formHooks) tea.Cmd {
	if h.ownKey != nil {
		if cmd, ok := h.ownKey(msg); ok {
			return cmd
		}
	}
	switch {
	case key.Matches(msg, s.keymap.Save):
		return s.save(h.save)
	case key.Matches(msg, s.keymap.Esc):
		return s.form.exitOrAsk(msgCmd(h.cancel), &s.toast)
	case s.openBorders(msg, h.borders):
		return nil
	}
	_, cmd := s.form.handleKey(msg)
	return cmd
}

// save ends insert mode and sends the screen's save message, or shows why
// the values cannot be saved.
func (s *formScreen) save(saveMsg func() (tea.Msg, error)) tea.Cmd {
	s.form.exitEdit()
	s.toast = toast{}
	msg, err := saveMsg()
	if err != nil {
		s.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	return msgCmd(msg)
}

// openBorders opens the border_fmt sub-modal for Enter, Space or l on that
// field, and reports whether it did.
func (s *formScreen) openBorders(msg tea.KeyMsg, borders func() *paneBorderFormatsModel) bool {
	if s.form.focusedFieldKey() != "border_fmt" || !slices.Contains([]string{"enter", " ", "l"}, msg.String()) {
		return false
	}
	s.borderPicker = borders()
	s.borderPicker.parentCrumb = s.parentCrumb
	s.sizeBorders()
	return true
}

// sizeBorders fits the open border_fmt sub-modal to the window.
func (s *formScreen) sizeBorders() {
	if s.borderPicker != nil {
		w, h := pickerModalSize(s.width, s.height)
		s.borderPicker.Update(tea.WindowSizeMsg{Width: w, Height: h})
	}
}

func (s *formScreen) refreshAccentStyles() {
	s.form.applyFocusStyles()
	if s.borderPicker != nil {
		s.borderPicker.refreshAccentStyles()
	}
}
