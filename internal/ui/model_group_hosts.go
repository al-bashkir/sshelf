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
	return []keyAction{
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
		{keys: km.ToggleSel, inList: true, run: func() tea.Cmd { m.toggleCurrentSelection(); return nil }},
		{keys: km.SelectAll, inList: true, run: func() tea.Cmd { m.selectAllFiltered(); return nil }},
		{keys: km.ClearSel, inList: true, run: func() tea.Cmd { m.clearSelection(); return nil }},
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
	}
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
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Group Hosts", m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		return renderCmdPromptModal(m.width, m.height, m.cmdPromptCrumb,
			"Connect and run a remote command (keeps session open).", m.cmdInput)
	}
	if m.confirmConnect {
		modal := connectConfirmBox(max(0, m.width-4), m.confirmConnectCount, m.confirmConnectHosts)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}
	if m.confirmRemove {
		innerW := max(0, m.width-2)
		innerH := max(0, m.height-2)
		contentH := max(0, innerH-6)
		modal := removeHostsConfirmBox(innerW, m.removeList, m.group.Name)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		breadcrumb := dim.Render("Groups >") + " " + headerStyle.Render(m.group.Name)
		right := statusDot(true, false)
		return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, searchHeader(&m.search, m.width, right), right, placed, "")
	}

	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(true, false)
		shown := len(m.list.Items())
		total := len(m.allHosts)
		q := strings.TrimSpace(m.search.Value())
		if q != "" {
			right += dim.Render(fmt.Sprintf(" %d / %d hosts", shown, total))
		} else {
			right += dim.Render(fmt.Sprintf(" %d hosts", total))
		}
		if selCount := len(m.selected); selCount > 0 {
			right += "   " + badgeSelStyle.Render(fmt.Sprintf("%d selected", selCount))
		}
	}
	var footer string
	if m.width < compactWidthThreshold {
		footer = styledFooter("\u21b5 connect  \u2423 select  esc back  ? help")
	} else {
		footer = styledFooter("\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  Ctrl+o cmd  a add")
		if m.height >= twoLineFooterMinHeight {
			footer += "\n" + styledFooter("e config  c custom  d remove  y copy  ·  Alt+1 hosts  Alt+3 settings  esc back  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	breadcrumb := dim.Render("Groups >") + " " + headerStyle.Render(m.group.Name)
	return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, searchHeader(&m.search, m.width, right), right, listContent, footer)
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

func (m *groupHostsModel) emptyStateView() string {
	return m.emptyView(
		dim.Render("No hosts in this group.") + "\n" + dim.Render("a \u2014 add hosts"))
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
