package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// listScreen is what the Hosts, Groups and Group hosts screens share: a
// searchable list with help, a remote-command prompt and connect actions.
// Each screen embeds it, so the fields read as m.list, m.search, ...
type listScreen struct {
	connectActions
	opts Options

	width  int
	height int

	keymap      keyMap
	list        list.Model
	search      textinput.Model
	focus       focusState
	prevSearch  string
	navPendingG bool

	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model
}

// newListScreen builds a list screen around l, with the list focused.
func newListScreen(opts Options, l list.Model) listScreen {
	configureList(&l)
	return listScreen{opts: opts, keymap: defaultKeyMap(), list: l, search: newSearchInput(), focus: focusList}
}

// update is the Update the list screens share: size the list, route keys
// (overlay, focus keys, then the screen's actions), show toasts, and give
// anything else to the search box or the list.
func (s *listScreen) update(msg tea.Msg, overlay func(tea.KeyMsg) (tea.Cmd, bool),
	actions func() []keyAction, applyFilter func(string)) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		s.list.SetSize(max(0, s.width-2), tabBoxListContentHeight(s.width, s.height)) // View sizes the search box
		return nil
	case tea.KeyMsg:
		if cmd, ok := routeListKey(msg, overlay, s.keymap, &s.focus, &s.search, actions); ok {
			return cmd
		}
	case toastMsg:
		return s.showToast(toast(msg), s.opts.Popup)
	}
	return updateSearchOrList(s.focus, &s.search, &s.list, &s.prevSearch, msg, applyFilter)
}

// acceptQuery handles Enter while the search box has focus (see
// acceptSearch). It reports false when the list has focus.
func (s *listScreen) acceptQuery(applyFilter func(string)) bool {
	if s.focus != focusSearch {
		return false
	}
	acceptSearch(&s.focus, &s.search, &s.prevSearch, len(s.list.Items()), applyFilter)
	return true
}

// openCmdPromptFor opens the remote-command prompt for targets, titled
// with prefix. With no targets it does nothing.
func (s *listScreen) openCmdPromptFor(prefix string, targets []string) tea.Cmd {
	if len(targets) == 0 {
		return nil
	}
	s.cmdPromptCrumb = cmdPromptCrumb(prefix, targets)
	s.cmdInput = newCmdPromptInput(s.width, s.height)
	s.cmdPrompt = true
	return nil
}

// emptyView fills the list area when no row shows: a no-match note while
// a search is typed, else hint.
func (s *listScreen) emptyView(hint string) string {
	return renderListEmptyState(s.width, s.height, s.search.Value(), hint)
}
