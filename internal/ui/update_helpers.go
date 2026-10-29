package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// setFocus moves focus between the search bar and the list, keeping the
// input's focus state and styling in sync.
func setFocus(focus *focusState, search *textinput.Model, to focusState) {
	*focus = to
	if to == focusSearch {
		search.Focus()
	} else {
		search.Blur()
	}
	setSearchBarFocused(search, to == focusSearch)
}

// handleFocusKeys handles the focus-switch keys shared by every list screen.
func handleFocusKeys(msg tea.KeyMsg, km keyMap, focus *focusState, search *textinput.Model) bool {
	switch {
	case key.Matches(msg, km.FocusSearch):
		setFocus(focus, search, focusSearch)
		return true
	case key.Matches(msg, km.ToggleFocus):
		if *focus == focusSearch {
			setFocus(focus, search, focusList)
		} else {
			setFocus(focus, search, focusSearch)
		}
		return true
	}
	return false
}

// handleEscChain runs the Esc priority shared by the list screens:
// blur search → clear selection → clear search. It returns false when none
// applied, leaving the caller to decide what Esc means next (e.g. go back).
// clearSel may be nil on screens without selection.
func handleEscChain(
	focus *focusState,
	search *textinput.Model,
	prevSearch *string,
	selectedCount int,
	clearSel func(),
	applyFilter func(string),
) bool {
	if *focus == focusSearch && search.Value() == "" {
		setFocus(focus, search, focusList)
		return true
	}
	if clearSel != nil && selectedCount > 0 {
		clearSel()
		return true
	}
	if search.Value() != "" {
		clearSearch(focus, search, prevSearch, applyFilter, false)
		return true
	}
	return false
}

// handlePickerEsc runs the Esc priority for the modal pickers: clear the
// search text (staying focused), then blur, then cancel. It returns false
// when the caller should close the picker.
func handlePickerEsc(focus *focusState, search *textinput.Model, prevSearch *string, applyFilter func(string)) bool {
	if search.Value() != "" {
		clearSearch(focus, search, prevSearch, applyFilter, *focus == focusSearch)
		return true
	}
	if *focus == focusSearch {
		setFocus(focus, search, focusList)
		return true
	}
	return false
}

// clearSearch empties the query and re-filters, optionally keeping focus.
func clearSearch(focus *focusState, search *textinput.Model, prevSearch *string, applyFilter func(string), keepFocus bool) {
	search.SetValue("")
	applyFilter("")
	*prevSearch = ""
	if !keepFocus && *focus == focusSearch {
		setFocus(focus, search, focusList)
	}
}

// acceptSearch is Enter while the search bar has focus: drop a query that
// matched nothing, then return to the list.
func acceptSearch(focus *focusState, search *textinput.Model, prevSearch *string, itemCount int, applyFilter func(string)) {
	if itemCount == 0 && search.Value() != "" {
		search.SetValue("")
		applyFilter("")
		*prevSearch = ""
	}
	setFocus(focus, search, focusList)
}

// cmdPromptCrumb builds the breadcrumb shown above the remote-command prompt.
func cmdPromptCrumb(prefix string, targets []string) string {
	if len(targets) == 1 {
		return prefix + " > " + targets[0]
	}
	return fmt.Sprintf("%s > %d selected", prefix, len(targets))
}

// newCmdPromptInput builds the remote-command input, sized for its modal.
func newCmdPromptInput(totalW, totalH int) textinput.Model {
	in := textinput.New()
	in.CharLimit = 512
	in.Prompt = "cmd: "
	in.Placeholder = "run on remote, keep session open"

	mw, mh := modalSize(totalW, totalH, cmdPromptMaxW, cmdPromptMaxH, cmdPromptMarginW, cmdPromptMarginH)
	innerW, _ := frameInnerSize(mw, mh)
	in.Width = min(70, max(1, innerW-len(in.Prompt)))

	in.Focus()
	configureSearch(&in)
	setSearchFocused(&in, true)
	return in
}

// handleCmdPromptKey drives the remote-command prompt. run is called with the
// entered command once the user confirms a non-empty line.
func handleCmdPromptKey(msg tea.KeyMsg, open *bool, in *textinput.Model, t *toast, run func(string) tea.Cmd) tea.Cmd {
	switch msg.String() {
	case "esc":
		*open = false
		in.Blur()
		return nil
	case "enter":
		cmd := strings.TrimSpace(in.Value())
		*open = false
		in.Blur()
		in.SetValue("")
		if cmd == "" {
			return nil
		}
		*t = toast{}
		return run(cmd)
	default:
		var teaCmd tea.Cmd
		*in, teaCmd = in.Update(msg)
		return teaCmd
	}
}

// handleConfirmQuit processes a key when the quit-confirm dialog is shown.
// It returns (true, cmd) if the key was consumed, (false, nil) otherwise.
// quitting may be nil for models that don't track a quitting state (e.g. pickers).
// clearToast controls whether the toast is cleared when the user cancels.
func handleConfirmQuit(
	msg tea.KeyMsg,
	confirmQuit *bool,
	t *toast,
	quitting *bool,
	clearToast bool,
) (handled bool, cmd tea.Cmd) {
	if !*confirmQuit {
		return false, nil
	}
	switch msg.String() {
	case "y", "Y", "enter":
		if quitting != nil {
			*quitting = true
		}
		return true, tea.Quit
	case "n", "N", "esc":
		*confirmQuit = false
		if clearToast {
			*t = toast{}
		}
		return true, nil
	default:
		return true, nil
	}
}

// handleConfirmConnect processes a key when the connect-confirm dialog is shown.
// Returns (true, cmd) if the key was consumed, (false, nil) otherwise.
func handleConfirmConnect(
	msg tea.KeyMsg,
	confirmConnect *bool,
	pendingFn *func() tea.Cmd,
	t *toast,
) (handled bool, cmd tea.Cmd) {
	if !*confirmConnect {
		return false, nil
	}
	switch msg.String() {
	case "y", "Y", "enter":
		*confirmConnect = false
		fn := *pendingFn
		*pendingFn = nil
		if fn != nil {
			return true, fn()
		}
		return true, nil
	case "n", "N", "esc":
		*confirmConnect = false
		*pendingFn = nil
		*t = toast{}
		return true, nil
	default:
		return true, nil
	}
}

func handleVimListNav(msg tea.KeyMsg, l *list.Model, pendingG *bool) bool {
	switch msg.String() {
	case "g":
		if *pendingG {
			start, end := l.Paginator.GetSliceBounds(len(l.VisibleItems()))
			if start < end {
				l.Select(start)
			}
			*pendingG = false
			return true
		}
		*pendingG = true
		return true
	case "G":
		start, end := l.Paginator.GetSliceBounds(len(l.VisibleItems()))
		if start < end {
			l.Select(end - 1)
		}
		*pendingG = false
		return true
	default:
		*pendingG = false
		return false
	}
}

// updateSearchOrList delegates a tea.Msg to either the search input or the
// list, depending on the current focus. When the search value changes,
// applyFn is called with the new query. Returns the resulting tea.Cmd.
func updateSearchOrList(
	focus focusState,
	search *textinput.Model,
	l *list.Model,
	prevSearch *string,
	msg tea.Msg,
	applyFn func(string),
) tea.Cmd {
	var cmd tea.Cmd
	if focus == focusSearch {
		*search, cmd = search.Update(msg)
		cur := search.Value()
		if cur != *prevSearch {
			applyFn(cur)
			*prevSearch = cur
		}
		return cmd
	}
	*l, cmd = l.Update(msg)
	return cmd
}
