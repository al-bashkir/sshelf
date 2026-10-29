package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// newSearchInput creates a pre-configured search textinput used by all list screens.
func newSearchInput() textinput.Model {
	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)
	return search
}

func configureList(m *list.Model) {
	// Avoid default letter shortcuts that conflict with our app keys.
	km := list.DefaultKeyMap()
	// Keep vim-style page keys (h/l) but drop other letters (b/f, etc.).
	km.NextPage.SetKeys("right", "pgdown", "l")
	km.PrevPage.SetKeys("left", "pgup", "h")
	km.GoToStart.SetKeys("home")
	km.GoToStart.SetHelp("home/gg", "page top")
	km.GoToEnd.SetKeys("end", "G")
	km.GoToEnd.SetHelp("end/G", "page bottom")
	m.KeyMap = km

	// We render our own header/footer.
	m.SetShowTitle(false)
	m.SetShowPagination(false)
	m.SetShowHelp(false)
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	m.DisableQuitKeybindings()
}
