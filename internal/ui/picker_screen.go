package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// pickerScreen is what the host and group pickers share: a searchable list
// in a centered frame, with a help modal. Each picker embeds it.
type pickerScreen struct {
	searchList
	parentCrumb string
	toast       toast
	topLines    []string // shown above the search line; set before a resize
}

// pickerHooks is what one picker adds to the shared flow.
type pickerHooks struct {
	cancel      tea.Msg        // Esc with nothing left to clear
	done        func() tea.Cmd // Enter in the list
	applyFilter func(string)   // shows the rows that match the search
	more        []keyAction    // the picker's own keys; may be nil
}

// newPickerScreen builds a picker around l, with the list focused.
func newPickerScreen(opts Options, l list.Model) pickerScreen {
	return pickerScreen{searchList: newSearchList(opts, l)}
}

// update sizes the list, routes keys (help, focus keys, then the actions)
// and gives anything else to the search box or the list.
func (s *pickerScreen) update(msg tea.Msg, h pickerHooks) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		innerW, innerH := frameInnerSize(msg.Width, msg.Height)
		s.list.SetSize(innerW, max(1, innerH-5-len(s.topLines)))
		s.search.Width = max(10, innerW-len(s.search.Prompt))
		return nil
	case tea.KeyMsg:
		actions := func() []keyAction { return s.actions(h) }
		if cmd, ok := routeListKey(msg, s.helpKey, s.keymap, &s.focus, &s.search, actions); ok {
			return cmd
		}
	}
	return updateSearchOrList(s.focus, &s.search, &s.list, &s.prevSearch, msg, h.applyFilter)
}

// helpKey keeps every key while help shows; ? or Esc closes it.
func (s *pickerScreen) helpKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !s.showHelp {
		return nil, false
	}
	if key.Matches(msg, s.keymap.Help) || msg.String() == "esc" {
		s.showHelp = false
	}
	return nil, true
}

// actions are the keys every picker has, then its own: ? opens help, Esc
// clears the search, leaves it, then cancels, and Enter leaves the search
// or is done.
func (s *pickerScreen) actions(h pickerHooks) []keyAction {
	return append([]keyAction{
		{keys: s.keymap.Help, run: func() tea.Cmd { s.showHelp = true; return nil }},
		{keys: s.keymap.Esc, run: func() tea.Cmd {
			if handlePickerEsc(&s.focus, &s.search, &s.prevSearch, h.applyFilter) {
				return nil
			}
			return msgCmd(h.cancel)
		}},
		{keys: s.keymap.Connect, run: func() tea.Cmd {
			if s.focus == focusSearch {
				setFocus(&s.focus, &s.search, focusList)
				return nil
			}
			return h.done()
		}},
	}, h.more...)
}

// view draws help when it shows, else the top lines, the search line and
// the list in a centered frame titled after the parent crumb.
func (s *pickerScreen) view(title, helpTitle string, help func() helpMap, status string) string {
	if s.showHelp {
		return renderHelpModal(s.width, s.height, helpTitle, help())
	}
	innerW, _ := frameInnerSize(s.width, s.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	parts := append(append([]string(nil), s.topLines...), s.search.View(), sep, strings.TrimRight(s.list.View(), "\n"), sep)
	body := strings.TrimRight(strings.Join(parts, "\n"), "\n")
	return renderFocusedFrame(s.width, s.height, breadcrumbTitle(s.parentCrumb, title), "", body, status)
}

func (s *pickerScreen) refreshAccentStyles() {
	setSearchBarFocused(&s.search, s.focus == focusSearch)
}
