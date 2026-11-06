package ui

import (
	"fmt"
	"io"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type groupsDelegate struct{}

func (d groupsDelegate) Height() int                             { return 1 }
func (d groupsDelegate) Spacing() int                            { return 0 }
func (d groupsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row := item.(groupRow)
	fmt.Fprint(w, renderGroupRow(m.Width(), index == m.Index(), row.name, row.hostCount, row.matchedIndexes))
}

type groupRow struct {
	index          int
	name           string
	hostCount      int
	matchedIndexes []int
}

func (i groupRow) Title() string       { return i.name }
func (i groupRow) Description() string { return "" }
func (i groupRow) FilterValue() string { return i.name }

type groupsModel struct {
	listScreen

	allRows []groupRow
	rows    []groupRow

	groupDelete
	confirmQuit bool
}

func newGroupsModel(opts Options) *groupsModel {
	rows := groupRows(opts.Inventory, "") // subgroups show inside their group
	items := make([]list.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, r)
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetDelegate(groupsDelegate{})
	l.Title = "Groups"
	return &groupsModel{listScreen: newListScreen(opts, l), allRows: rows, rows: rows}
}

func (m *groupsModel) setRows(rows []groupRow) {
	m.rows = rows
	items := make([]list.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, r)
	}
	m.list.SetItems(items)
}

func (m *groupsModel) Refresh(inv config.Inventory) {
	m.opts.Inventory = inv
	m.allRows = groupRows(inv, "")
	m.applyFilter(m.search.Value())
}

func (m *groupsModel) Init() tea.Cmd { return nil }

func (m *groupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.update(msg, m.overlayKey, m.keyActions, m.applyFilter)
}

// overlayKey handles keys while help, the command prompt or a question
// shows, and vim g/gg/G in the list.
func (m *groupsModel) overlayKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case m.showHelp:
		helpViewKey(msg, m.keymap.Help, &m.showHelp, &m.helpVP)
		return nil, true
	case m.cmdPrompt:
		run := func(cmd string) tea.Cmd { return m.connectAllCmd(false, cmd) }
		return handleCmdPromptKey(msg, &m.cmdPrompt, &m.cmdInput, &m.toast, run), true
	}
	if handled, cmd := handleConfirmQuit(msg, &m.confirmQuit, &m.toast, &m.quitting, true); handled {
		return cmd, true
	}
	if m.confirmDelete {
		return m.deleteKey(msg, &m.toast), true
	}
	if handled, cmd := handleConfirmConnect(msg, &m.confirmConnect, &m.pendingConnectFn, &m.toast); handled {
		return cmd, true
	}
	return nil, m.focus == focusList && handleVimListNav(msg, &m.list, &m.navPendingG)
}

func (m *groupsModel) keyActions() []keyAction {
	km := m.keymap
	return []keyAction{
		{keys: km.HostsTab, run: func() tea.Cmd { return msgCmd(switchScreenMsg{to: screenHosts}) }},
		{keys: km.Settings, run: func() tea.Cmd { return msgCmd(openDefaultsFormMsg{returnTo: screenGroups}) }},
		{keys: km.Quit, inList: true, run: func() tea.Cmd {
			return askQuit(m.opts.Config.Defaults.ConfirmQuit, &m.confirmQuit, &m.toast, &m.quitting)
		}},
		{keys: km.Help, run: func() tea.Cmd {
			toggleHelpView(&m.showHelp, &m.helpVP, m.width, m.height, "Groups", m.helpKeys)
			return nil
		}},
		{keys: km.Esc, run: func() tea.Cmd {
			handleEscChain(&m.focus, &m.search, &m.prevSearch, 0, nil, m.applyFilter)
			return nil
		}},
		{keys: km.NewGroup, inList: true, run: func() tea.Cmd { return msgCmd(openGroupFormMsg{index: -1, returnTo: screenGroups}) }},
		{keys: km.EditGroup, inList: true, run: m.onRow(func(r groupRow) tea.Cmd {
			return msgCmd(openGroupFormMsg{index: r.index, returnTo: screenGroups})
		})},
		{keys: km.AddHosts, inList: true, run: m.onRow(func(r groupRow) tea.Cmd {
			return msgCmd(openHostPickerMsg{groupIndex: r.index, returnTo: screenGroups})
		})},
		{keys: km.CustomHost, inList: true, run: m.onRow(func(r groupRow) tea.Cmd {
			return msgCmd(openCustomHostMsg{returnTo: screenGroups, groupIndex: r.index})
		})},
		{keys: km.DeleteGroup, inList: true, run: m.onRow(func(r groupRow) tea.Cmd { return m.askDelete(r, &m.toast) })},
		{keys: km.Connect, run: m.enter},
		{keys: km.ConnectCmd, inList: true, run: m.onRow(m.openCmdPrompt)},
		{keys: km.ConnectAll, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connectAllCmd(false, "")
		}},
		{keys: km.Copy, inList: true, run: m.copyGroup},
		{keys: km.OneWindow, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connectAllCmd(true, "")
		}},
	}
}

// onRow returns an action that runs f on the selected group; with no
// group selected it does nothing.
func (m *groupsModel) onRow(f func(groupRow) tea.Cmd) func() tea.Cmd {
	return func() tea.Cmd {
		row, ok := m.list.SelectedItem().(groupRow)
		if !ok {
			return nil
		}
		return f(row)
	}
}

// enter accepts the query in the search box, or opens the group from the
// list.
func (m *groupsModel) enter() tea.Cmd {
	if m.acceptQuery(m.applyFilter) {
		return nil
	}
	return m.onRow(func(r groupRow) tea.Cmd { return msgCmd(openGroupHostsMsg{index: r.index}) })()
}

func (m *groupsModel) openCmdPrompt(r groupRow) tea.Cmd {
	return m.openCmdPromptTitled("Groups > " + r.name)
}

// copyGroup opens a new-group form filled from the selected group.
func (m *groupsModel) copyGroup() tea.Cmd {
	row, ok := m.list.SelectedItem().(groupRow)
	if !ok || row.index >= len(m.opts.Inventory.Groups) {
		m.toast = toast{text: "no group selected", level: toastWarn}
		return nil
	}
	return copyGroupCmd(m.opts.Inventory, row.index, screenGroups)
}

func (m *groupsModel) View() string {
	if v, ok := m.overlayView("Groups", m.helpKeys, "Connect and run a remote command for all hosts (keeps sessions open)."); ok {
		return v
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	count := statusDot(false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
	if m.confirmDelete {
		return renderMainTabBox(m.width, m.height, 1, searchHeader(&m.search, m.width, count), count, m.deleteView())
	}
	right := count
	if !m.toast.empty() {
		right = renderToast(m.toast)
	}
	footer := m.hintFooter("\u21b5 open  C connect  ? help",
		"\u21b5 open  C connect  ·  o panes  Ctrl+o cmd  ·  n new",
		"e edit  d delete  y copy  a add hosts  c custom  ·  Alt+1 hosts  Alt+3 settings  tab search  ? help")
	return renderMainTabBoxWithFooter(m.width, m.height, 1, searchHeader(&m.search, m.width, right), right,
		m.listOrEmpty(func() string { return m.emptyView("No groups yet.", "n \u2014 create a new group") }), footer)
}

// deleteView centers the delete question in the list area.
func (m *groupsModel) deleteView() string {
	innerW := max(0, m.width-2)
	contentH := max(0, max(0, m.height-2)-4)
	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, m.deleteBox(m.opts.Inventory.Groups, innerW))
}

func (m *groupsModel) helpKeys() helpMap {
	openGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open group"),
	)

	return helpMap{
		sections: []helpSection{
			{title: "Navigation", keys: []key.Binding{
				m.list.KeyMap.CursorUp,
				m.list.KeyMap.CursorDown,
				m.list.KeyMap.PrevPage,
				m.list.KeyMap.NextPage,
				m.list.KeyMap.GoToStart,
				m.list.KeyMap.GoToEnd,
				m.keymap.FocusSearch,
				m.keymap.ToggleFocus,
				m.keymap.HostsTab,
				m.keymap.Settings,
				m.keymap.Esc,
			}},
			{title: "Connection", keys: []key.Binding{
				openGroup,
				m.keymap.ConnectAll,
				m.keymap.ConnectCmd,
				m.keymap.OneWindow,
				m.keymap.CustomHost,
			}},
			{title: "Editing", keys: []key.Binding{
				m.keymap.NewGroup,
				m.keymap.EditGroup,
				m.keymap.DeleteGroup,
				m.keymap.AddHosts,
				m.keymap.Copy,
			}},
			{title: "General", keys: []key.Binding{
				m.keymap.Settings,
				m.keymap.Help,
				m.keymap.Quit,
			}},
		},
	}
}

func (m *groupsModel) applyFilter(query string) {
	var prevName string
	prevIndex := -1
	if row, ok := m.list.SelectedItem().(groupRow); ok {
		prevName = row.name
		prevIndex = row.index
	}

	rows := matchGroupRows(m.allRows, query)
	m.setRows(rows)

	if len(rows) == 0 {
		return
	}
	// Prefer index-based restore: stable across renames.
	if prevIndex >= 0 {
		for i, r := range rows {
			if r.index == prevIndex {
				m.list.Select(i)
				return
			}
		}
	}
	restoreCursor(&m.list, groupNames(m.allRows), groupNames(rows), prevName)
}

func groupNames(rows []groupRow) []string {
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.name)
	}
	return names
}

func (m *groupsModel) connectAllCmd(oneWindow bool, remoteCmd string) tea.Cmd {
	row, ok := m.list.SelectedItem().(groupRow)
	if !ok {
		m.toast = toast{text: "no group selected", level: toastWarn}
		return nil
	}
	return m.connectGroup(m.opts, row.index, oneWindow, remoteCmd)
}

func (m *groupsModel) refreshAccentStyles() {
	restyleListInputs(&m.search, &m.cmdInput, m.focus == focusSearch, m.cmdPrompt)
}
