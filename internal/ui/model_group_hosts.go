package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/sshcmd"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type groupHostsModel struct {
	hostSelectList
	listScreen

	groupIndex int
	group      config.Group

	confirmRemove bool
	removeList    []string
}

func newGroupHostsModel(opts Options, groupIndex int) *groupHostsModel {
	g := config.Group{}
	if groupIndex >= 0 && groupIndex < len(opts.Inventory.Groups) {
		g = opts.Inventory.Groups[groupIndex]
	}

	items := make([]list.Item, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		_, ok := sshcmd.FindHostConfig(opts.Inventory.Hosts, h)
		items = append(items, hostRow{host: h, hasCfg: ok})
	}

	l := list.New(items, hostDelegate{}, 0, 0)
	l.Title = "Group: " + g.Name
	m := &groupHostsModel{
		hostSelectList: newHostSelectList(g.Hosts),
		listScreen:     newListScreen(opts, l),
		groupIndex:     groupIndex,
		group:          g,
	}
	m.bindList(&m.list)
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
		{keys: km.DeleteGroup, inList: true, run: m.askRemove},
		{keys: km.Esc, run: m.back},
		{keys: km.Quit, inList: true, run: m.back},
		{keys: km.ConnectCmd, inList: true, run: func() tea.Cmd {
			return m.openCmdPromptFor("Groups > "+m.group.Name, m.hostsToOpen())
		}},
		{keys: km.Connect, run: m.enter},
		{keys: km.OneWindow, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connect(m.hostsToOpen(), m.opts, &m.group, true, "")
		}},
		{keys: km.ConnectSame, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connectSameWindow(m.hostsToOpen(), m.opts, &m.group)
		}},
		{keys: km.AddHosts, inList: true, run: func() tea.Cmd {
			return msgCmd(openHostPickerMsg{groupIndex: m.groupIndex, returnTo: screenGroupHosts})
		}},
		{keys: km.CustomHost, inList: true, run: func() tea.Cmd {
			return msgCmd(openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: m.groupIndex})
		}},
		{keys: km.HostConfig, inList: true, run: func() tea.Cmd {
			return openHostConfigCmd(m.currentHost(), screenGroupHosts, &m.toast)
		}},
		{keys: km.Copy, inList: true, run: func() tea.Cmd {
			return copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenGroupHosts, &m.toast)
		}},
	}...)
}

// back is Esc, or q in the list: blur search, clear the selection, clear
// the search, and only then go back to Groups.
func (m *groupHostsModel) back() tea.Cmd {
	if handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter) {
		return nil
	}
	return msgCmd(switchScreenMsg{to: screenGroups})
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

// enter accepts the query in the search box, or connects from the list.
func (m *groupHostsModel) enter() tea.Cmd {
	if m.acceptQuery(m.applyFilter) {
		return nil
	}
	m.toast = toast{}
	return m.connect(m.hostsToOpen(), m.opts, &m.group, false, "")
}

func (m *groupHostsModel) View() string {
	if v, ok := m.overlayView("Group Hosts", m.helpKeys, "Connect and run a remote command (keeps session open)."); ok {
		return v
	}
	crumb := dim.Render("Groups >") + " " + headerStyle.Render(m.group.Name)
	if m.confirmRemove {
		dot := statusDot(false)
		return renderBreadcrumbTabBox(m.width, m.height, crumb, searchHeader(&m.search, m.width, dot), dot, m.removeView(), "")
	}
	right := m.status()
	footer := m.hintFooter("\u21b5 connect  \u2423 select  esc back  ? help",
		"\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  Ctrl+o cmd  a add",
		"e config  c custom  d remove  y copy  ·  Alt+1 hosts  Alt+3 settings  esc back  ? help")
	return renderBreadcrumbTabBox(m.width, m.height, crumb, searchHeader(&m.search, m.width, right), right,
		m.listOrEmpty(func() string { return m.emptyView("No hosts in this group.", "a \u2014 add hosts") }), footer)
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

// removeView centers the remove question in the list area.
func (m *groupHostsModel) removeView() string {
	innerW := max(0, m.width-2)
	contentH := max(0, max(0, m.height-2)-6)
	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, removeHostsConfirmBox(innerW, m.removeList, m.group.Name))
}

func (m *groupHostsModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	remove := key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "remove"),
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
				m.keymap.Connect,
				m.keymap.ConnectSame,
				m.keymap.ConnectCmd,
				m.keymap.OneWindow,
			}},
			{title: "Editing", keys: []key.Binding{
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

func (m *groupHostsModel) applyFilter(query string) {
	m.filter(query, m.opts.Inventory, false, false)
}

func (m *groupHostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	if strings.TrimSpace(remoteCmd) == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}
	return m.connect(m.hostsToOpen(), m.opts, &m.group, false, remoteCmd)
}

func (m *groupHostsModel) refreshAccentStyles() {
	restyleListInputs(&m.search, &m.cmdInput, m.focus == focusSearch, m.cmdPrompt)
}
