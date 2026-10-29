package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit        key.Binding
	Help        key.Binding
	FocusSearch key.Binding
	ToggleFocus key.Binding
	HostsTab    key.Binding
	GroupsTab   key.Binding
	Reload      key.Binding
	Esc         key.Binding
	Settings    key.Binding
	Save        key.Binding
	CustomHost  key.Binding
	HostConfig  key.Binding
	ConnectCmd  key.Binding
	ConnectSame key.Binding
	ToggleSel   key.Binding
	SelectAll   key.Binding
	ClearSel    key.Binding
	Connect     key.Binding
	ConnectAll  key.Binding
	OneWindow   key.Binding
	NewGroup    key.Binding
	EditGroup   key.Binding
	DeleteGroup key.Binding
	AddHosts    key.Binding
	Copy        key.Binding
	HideHost    key.Binding
	ShowHidden  key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		FocusSearch: key.NewBinding(
			key.WithKeys("ctrl+f", "/"),
			key.WithHelp("ctrl+f /", "search"),
		),
		ToggleFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "search/list"),
		),
		HostsTab: key.NewBinding(
			key.WithKeys("alt+1"),
			key.WithHelp("alt+1", "hosts"),
		),
		GroupsTab: key.NewBinding(
			key.WithKeys("alt+2"),
			key.WithHelp("alt+2", "groups"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear/blur"),
		),
		Settings: key.NewBinding(
			key.WithKeys("alt+3"),
			key.WithHelp("alt+3", "settings"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		CustomHost: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "custom host"),
		),
		HostConfig: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "config"),
		),
		ConnectCmd: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "connect with custom command"),
		),
		ConnectSame: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "open in current pane"),
		),
		ToggleSel: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		ClearSel: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "clear"),
		),
		Connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		ConnectAll: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "connect all"),
		),
		OneWindow: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "connect"),
		),
		NewGroup: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("n", "new"),
		),
		EditGroup: key.NewBinding(
			key.WithKeys("e", "E"),
			key.WithHelp("e", "edit"),
		),
		DeleteGroup: key.NewBinding(
			key.WithKeys("d", "D", "delete"),
			key.WithHelp("d", "delete"),
		),
		AddHosts: key.NewBinding(
			key.WithKeys("a", "A"),
			key.WithHelp("a", "add hosts to group"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		HideHost: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("Ctrl+H", "hide/unhide"),
		),
		ShowHidden: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "show hidden"),
		),
	}
}
