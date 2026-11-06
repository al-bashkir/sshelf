package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listScreen is what the Hosts, Groups and Group hosts screens share: a
// searchable list with help, a remote-command prompt and connect actions.
// Each screen embeds it, so the fields read as m.list, m.search, ...
type listScreen struct {
	connectActions
	searchList
	navPendingG bool

	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model
}

// newListScreen builds a list screen around l, with the list focused.
func newListScreen(opts Options, l list.Model) listScreen {
	return listScreen{searchList: newSearchList(opts, l)}
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
	return s.openCmdPromptTitled(cmdPromptCrumb(prefix, targets))
}

// openCmdPromptTitled opens the remote-command prompt titled title.
func (s *listScreen) openCmdPromptTitled(title string) tea.Cmd {
	s.cmdPromptCrumb = title
	s.cmdInput = newCmdPromptInput(s.width, s.height)
	s.cmdPrompt = true
	return nil
}

// emptyView fills the list area when no row shows: a no-match note while
// a search is typed, else the hint lines, dimmed.
func (s *listScreen) emptyView(hint ...string) string {
	for i, h := range hint {
		hint[i] = dim.Render(h)
	}
	return renderListEmptyState(s.width, s.height, s.search.Value(), strings.Join(hint, "\n"))
}

// overlayView draws help, the command prompt or the connect question when
// one shows; cmdHint is the prompt's help line.
func (s *listScreen) overlayView(helpTitle string, help func() helpMap, cmdHint string) (string, bool) {
	switch {
	case s.showHelp:
		return renderHelpModalWithVP(s.width, s.height, helpTitle, help(), &s.helpVP), true
	case s.cmdPrompt:
		return renderCmdPromptModal(s.width, s.height, s.cmdPromptCrumb, cmdHint, s.cmdInput), true
	case s.confirmConnect:
		box := connectConfirmBox(max(0, s.width-4), s.confirmConnectCount, s.confirmConnectHosts)
		return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, box), true
	}
	return "", false
}

// hintFooter is the compact hint line in a narrow window, else the wide
// line and, in a tall enough window, the second line.
func (s *listScreen) hintFooter(compact, wide, second string) string {
	switch {
	case s.width < compactWidthThreshold:
		return styledFooter(compact)
	case s.height < twoLineFooterMinHeight:
		return styledFooter(wide)
	}
	return styledFooter(wide + "\n" + second)
}

// listOrEmpty is the list, or empty() when no row shows.
func (s *listScreen) listOrEmpty(empty func() string) string {
	if len(s.list.Items()) == 0 {
		return empty()
	}
	return s.list.View()
}

// hostCount is " N hosts", or " shown / N hosts" while a search is typed.
func (s *listScreen) hostCount(total int) string {
	if strings.TrimSpace(s.search.Value()) != "" {
		return dim.Render(fmt.Sprintf(" %d / %d hosts", len(s.list.Items()), total))
	}
	return dim.Render(fmt.Sprintf(" %d hosts", total))
}
