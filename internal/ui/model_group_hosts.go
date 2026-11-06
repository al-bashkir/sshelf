package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// groupHostsModel shows one group like a folder: its subgroups first, as
// group rows, then its hosts.
type groupHostsModel struct {
	hostSelectList
	listScreen
	groupDelete

	groupIndex int
	group      config.Group
	subgroups  []groupRow

	confirmRemove bool
	removeList    []string
}

func newGroupHostsModel(opts Options, groupIndex int) *groupHostsModel {
	g := config.Group{}
	var subgroups []groupRow
	if groupIndex >= 0 && groupIndex < len(opts.Inventory.Groups) {
		g = opts.Inventory.Groups[groupIndex]
		subgroups = groupRows(opts.Inventory, config.GroupPath(g))
	}

	l := list.New(nil, hostDelegate{}, 0, 0)
	l.Title = "Group: " + g.Name
	m := &groupHostsModel{
		hostSelectList: newHostSelectList(g.Hosts),
		listScreen:     newListScreen(opts, l),
		groupIndex:     groupIndex,
		group:          g,
		subgroups:      subgroups,
	}
	m.bindList(&m.list)
	m.applyFilter("")
	return m
}

func (m *groupHostsModel) Init() tea.Cmd { return nil }

func (m *groupHostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.update(msg, m.overlayKey, m.keyActions, m.applyFilter)
}

// overlayKey handles keys while help, the command prompt or a question
// shows, and vim g/gg/G in the list.
func (m *groupHostsModel) overlayKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case m.showHelp:
		helpViewKey(msg, m.keymap.Help, &m.showHelp, &m.helpVP)
		return nil, true
	case m.cmdPrompt:
		return handleCmdPromptKey(msg, &m.cmdPrompt, &m.cmdInput, &m.toast, m.handleConnectWithRemoteCommand), true
	case m.confirmRemove:
		return m.removeKey(msg), true
	case m.confirmDelete:
		return m.deleteKey(msg, &m.toast), true
	}
	if handled, cmd := handleConfirmConnect(msg, &m.confirmConnect, &m.pendingConnectFn, &m.toast); handled {
		return cmd, true
	}
	return nil, m.focus == focusList && handleVimListNav(msg, &m.list, &m.navPendingG)
}

// removeKey answers the remove question: y removes, n cancels, and any
// other key leaves the question open.
func (m *groupHostsModel) removeKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y", "enter":
		toRemove := m.removeList
		m.confirmRemove, m.removeList = false, nil
		return msgCmd(removeHostsMsg{groupIndex: m.groupIndex, hosts: toRemove})
	case "n", "N", "esc":
		m.confirmRemove, m.removeList = false, nil
		m.toast = toast{}
	}
	return nil
}

func (m *groupHostsModel) keyActions() []keyAction {
	km := m.keymap
	return append(m.selectionActions(km), []keyAction{
		{keys: km.HostsTab, run: func() tea.Cmd { return msgCmd(switchScreenMsg{to: screenHosts}) }},
		{keys: km.GroupsTab, run: func() tea.Cmd { return msgCmd(switchScreenMsg{to: screenGroups}) }},
		{keys: km.Settings, run: func() tea.Cmd { return msgCmd(openDefaultsFormMsg{returnTo: screenGroupHosts}) }},
		{keys: km.Help, run: func() tea.Cmd {
			toggleHelpView(&m.showHelp, &m.helpVP, m.width, m.height, "Group Hosts", m.helpKeys)
			return nil
		}},
		{keys: km.DeleteGroup, inList: true, run: m.byRow(func(r groupRow) tea.Cmd { return m.askDelete(r, &m.toast) }, m.askRemove)},
		{keys: km.Esc, run: m.back},
		{keys: km.Quit, inList: true, run: m.back},
		{keys: km.ConnectCmd, inList: true, run: m.byTarget(
			func(r groupRow) tea.Cmd { return m.openCmdPromptTitled(groupTrail(m.opts.Inventory.Groups[r.index])) },
			func() tea.Cmd { return m.openCmdPromptFor(groupTrail(m.group), m.hostsToOpen()) })},
		{keys: km.Connect, run: m.enter},
		{keys: km.OneWindow, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.byTarget(
				func(r groupRow) tea.Cmd { return m.connectGroup(m.opts, r.index, true, "") },
				func() tea.Cmd { return m.connect(m.hostsToOpen(), m.opts, &m.group, true, "") })()
		}},
		{keys: km.ConnectAll, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.byRow(
				func(r groupRow) tea.Cmd { return m.connectGroup(m.opts, r.index, false, "") },
				func() tea.Cmd { return m.connectGroup(m.opts, m.groupIndex, false, "") })()
		}},
		{keys: km.ConnectSame, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connectSameWindow(m.hostsToOpen(), m.opts, &m.group)
		}},
		{keys: km.NewGroup, inList: true, run: func() tea.Cmd {
			return msgCmd(openGroupFormPrefillMsg{group: config.Group{Parent: config.GroupPath(m.group)}, returnTo: screenGroupHosts})
		}},
		{keys: km.AddHosts, inList: true, run: func() tea.Cmd {
			return msgCmd(openHostPickerMsg{groupIndex: m.groupIndex, returnTo: screenGroupHosts})
		}},
		{keys: km.CustomHost, inList: true, run: func() tea.Cmd {
			return msgCmd(openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: m.groupIndex})
		}},
		{keys: km.HostConfig, inList: true, run: m.byRow(
			func(r groupRow) tea.Cmd { return msgCmd(openGroupFormMsg{index: r.index, returnTo: screenGroupHosts}) },
			func() tea.Cmd { return openHostConfigCmd(m.currentHost(), screenGroupHosts, &m.toast) })},
		{keys: km.Copy, inList: true, run: m.byRow(
			func(r groupRow) tea.Cmd { return copyGroupCmd(m.opts.Inventory, r.index, screenGroupHosts) },
			func() tea.Cmd {
				return copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenGroupHosts, &m.toast)
			})},
	}...)
}

// byRow returns an action that runs onGroup on the subgroup under the
// cursor, and onHost on any other row.
func (m *groupHostsModel) byRow(onGroup func(groupRow) tea.Cmd, onHost func() tea.Cmd) func() tea.Cmd {
	return func() tea.Cmd {
		if r, ok := m.list.SelectedItem().(groupRow); ok {
			return onGroup(r)
		}
		return onHost()
	}
}

// byTarget is byRow for the connect keys: selected hosts win over the
// subgroup under the cursor.
func (m *groupHostsModel) byTarget(onGroup func(groupRow) tea.Cmd, onHosts func() tea.Cmd) func() tea.Cmd {
	return func() tea.Cmd {
		if r, ok := m.list.SelectedItem().(groupRow); ok && len(m.selected) == 0 {
			return onGroup(r)
		}
		return onHosts()
	}
}

// back is Esc, or q in the list: blur search, clear the selection, clear
// the search, and only then go up: to the parent group with the cursor on
// this one, or to Groups.
func (m *groupHostsModel) back() tea.Cmd {
	if handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter) {
		return nil
	}
	if m.group.Parent == "" {
		return msgCmd(switchScreenMsg{to: screenGroups})
	}
	parent := config.FindGroup(m.opts.Inventory.Groups, m.group.Parent)
	return msgCmd(openGroupHostsMsg{index: parent, from: config.GroupPath(m.group)})
}

// focusGroup puts the cursor on the row of subgroup i, or on the first row
// when it does not show.
func (m *groupHostsModel) focusGroup(i int) {
	m.list.Select(max(0, slices.IndexFunc(m.list.Items(), func(it list.Item) bool { r, ok := it.(groupRow); return ok && r.index == i })))
}

func (m *groupHostsModel) askRemove() tea.Cmd {
	toRemove := m.hostsToOpen()
	if len(toRemove) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	m.confirmRemove, m.removeList = true, toRemove
	m.toast = toast{text: fmt.Sprintf("remove %d? (y/n)", len(toRemove)), level: toastWarn}
	return nil
}

// enter accepts the query in the search box, opens the subgroup under the
// cursor, or connects from the list.
func (m *groupHostsModel) enter() tea.Cmd {
	if m.acceptQuery(m.applyFilter) {
		return nil
	}
	return m.byRow(
		func(r groupRow) tea.Cmd { return msgCmd(openGroupHostsMsg{index: r.index}) },
		func() tea.Cmd {
			m.toast = toast{}
			return m.connect(m.hostsToOpen(), m.opts, &m.group, false, "")
		})()
}

func (m *groupHostsModel) View() string {
	if v, ok := m.overlayView("Group Hosts", m.helpKeys, "Connect and run a remote command (keeps session open)."); ok {
		return v
	}
	crumb := breadcrumbTitle(strings.TrimSuffix(groupTrail(m.group), " > "+m.group.Name), m.group.Name)
	if m.confirmRemove || m.confirmDelete {
		dot := statusDot(false)
		return renderBreadcrumbTabBox(m.width, m.height, crumb, searchHeader(&m.search, m.width, dot), dot, m.questionView(), "")
	}
	right := m.status()
	footer := m.hintFooter("↵ open/connect  ␣ select  esc back  ? help",
		"↵ open/connect  O pane  ·  ␣ select  o panes  C all  ·  Ctrl+o cmd  a add",
		"e edit  c custom  d delete  y copy  n new group  ·  Alt+1 hosts  Alt+3 settings  esc back  ? help")
	return renderBreadcrumbTabBox(m.width, m.height, crumb, searchHeader(&m.search, m.width, right), right,
		m.listOrEmpty(func() string { return m.emptyView("No hosts in this group.", "a — add hosts") }), footer)
}

// status is the header's right side: a toast, or the host counts and the
// selection.
func (m *groupHostsModel) status() string {
	if !m.toast.empty() {
		return renderToast(m.toast)
	}
	right := statusDot(false) + m.hostCount(len(m.allHosts))
	if n := len(m.selected); n > 0 {
		right += "   " + badgeSelStyle.Render(fmt.Sprintf("%d selected", n))
	}
	return right
}

// questionView centers the open question, remove or delete, in the list area.
func (m *groupHostsModel) questionView() string {
	innerW := max(0, m.width-2)
	contentH := max(0, max(0, m.height-2)-6)
	box := removeHostsConfirmBox(innerW, m.removeList, m.group.Name)
	if m.confirmDelete {
		box = m.deleteBox(m.opts.Inventory.Groups, innerW)
	}
	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, box)
}

func (m *groupHostsModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	remove := key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "remove host / delete group"),
	)
	open := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open group / connect"),
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
				m.keymap.HostsTab,
				m.keymap.GroupsTab,
				m.keymap.Settings,
				m.keymap.FocusSearch,
				m.keymap.ToggleFocus,
				esc,
			}},
			{title: "Selection", keys: []key.Binding{
				m.keymap.ToggleSel,
				m.keymap.SelectAll,
				m.keymap.ClearSel,
			}},
			{title: "Connection", keys: []key.Binding{
				open,
				m.keymap.ConnectAll,
				m.keymap.ConnectSame,
				m.keymap.ConnectCmd,
				m.keymap.OneWindow,
			}},
			{title: "Editing", keys: []key.Binding{
				m.keymap.NewGroup,
				m.keymap.AddHosts,
				m.keymap.CustomHost,
				m.keymap.HostConfig,
				m.keymap.Copy,
				remove,
			}},
			{title: "General", keys: []key.Binding{
				m.keymap.Help,
			}},
		},
	}
}

// applyFilter filters the subgroups and the hosts; the subgroups come
// first. The cursor stays on its row while that row shows.
func (m *groupHostsModel) applyFilter(query string) {
	prev := m.list.SelectedItem()
	m.filter(query, m.opts.Inventory, false, false) // the host rows, cursor on the same host
	groups := matchGroupRows(m.subgroups, query)
	hostAt := m.list.Index()
	items := make([]list.Item, 0, len(groups)+len(m.list.Items()))
	for _, r := range groups {
		items = append(items, r)
	}
	items = append(items, m.list.Items()...)
	m.list.SetItems(items)
	switch p := prev.(type) {
	case hostRow:
		m.list.Select(max(0, min(len(groups)+hostAt, len(items)-1)))
	case groupRow:
		m.list.Select(max(0, slices.IndexFunc(groups, func(r groupRow) bool { return r.index == p.index })))
	}
}

func (m *groupHostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	if strings.TrimSpace(remoteCmd) == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}
	return m.byTarget(
		func(r groupRow) tea.Cmd { return m.connectGroup(m.opts, r.index, false, remoteCmd) },
		func() tea.Cmd { return m.connect(m.hostsToOpen(), m.opts, &m.group, false, remoteCmd) })()
}

func (m *groupHostsModel) refreshAccentStyles() {
	restyleListInputs(&m.search, &m.cmdInput, m.focus == focusSearch, m.cmdPrompt)
}
