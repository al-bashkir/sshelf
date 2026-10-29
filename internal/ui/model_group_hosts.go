package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/sshcmd"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type groupHostsModel struct {
	hostSelectList
	connectActions
	opts Options

	width  int
	height int

	groupIndex int
	group      config.Group

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap         keyMap
	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model

	confirmRemove bool
	removeList    []string

	prevSearch  string
	navPendingG bool
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
	configureList(&l)

	search := newSearchInput()

	m := &groupHostsModel{
		hostSelectList: newHostSelectList(g.Hosts),
		opts:           opts,
		groupIndex:     groupIndex,
		group:          g,
		list:           l,
		search:         search,
		focus:          focusList,
		keymap:         defaultKeyMap(),
	}
	m.bindList(&m.list)
	return m
}

func (m *groupHostsModel) Init() tea.Cmd { return nil }

func (m *groupHostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW := max(0, w-2)
		m.list.SetSize(innerW, tabBoxListContentHeight(w, h))
		promptW := len(m.search.Prompt)
		reserve := 24
		m.search.Width = max(10, innerW-reserve-promptW)
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}
			updateHelpViewport(&m.helpVP, msg)
			return m, nil
		}

		if m.cmdPrompt {
			return m, handleCmdPromptKey(msg, &m.cmdPrompt, &m.cmdInput, &m.toast, m.handleConnectWithRemoteCommand)
		}

		if m.confirmRemove {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				toRemove := append([]string(nil), m.removeList...)
				m.confirmRemove = false
				m.removeList = nil
				return m, func() tea.Msg { return removeHostsMsg{groupIndex: m.groupIndex, hosts: toRemove} }
			case "n", "N", "esc":
				m.confirmRemove = false
				m.removeList = nil
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}
		if handled, cmd := handleConfirmConnect(msg, &m.confirmConnect, &m.pendingConnectFn, &m.toast); handled {
			return m, cmd
		}

		if m.focus == focusList {
			if handleVimListNav(msg, &m.list, &m.navPendingG) {
				return m, nil
			}
		} else {
			m.navPendingG = false
		}

		if key.Matches(msg, m.keymap.HostsTab) {
			return m, func() tea.Msg { return switchScreenMsg{to: screenHosts} }
		}
		if key.Matches(msg, m.keymap.GroupsTab) {
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
		}
		if key.Matches(msg, m.keymap.Settings) {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenGroupHosts} }
		}

		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp && m.width > 0 && m.height > 0 {
				m.helpVP = initHelpViewport(m.width, m.height, "Group Hosts", m.helpKeys())
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.DeleteGroup) && m.focus == focusList {
			toRemove := m.hostsToOpen()
			if len(toRemove) == 0 {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			m.confirmRemove = true
			m.removeList = toRemove
			m.toast = toast{text: fmt.Sprintf("remove %d? (y/n)", len(toRemove)), level: toastWarn}
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) || (key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch) {
			// Esc/q priority: blur search → clear selection → clear search → back.
			if handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter) {
				return m, nil
			}
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
		}

		if key.Matches(msg, m.keymap.ToggleSel) && m.focus == focusList {
			m.toggleCurrentSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.SelectAll) && m.focus == focusList {
			m.selectAllFiltered()
			return m, nil
		}
		if key.Matches(msg, m.keymap.ClearSel) && m.focus == focusList {
			m.clearSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.ConnectCmd) && m.focus == focusList {
			targets := m.hostsToOpen()
			if len(targets) == 0 {
				return m, nil
			}
			m.cmdPromptCrumb = cmdPromptCrumb("Groups > "+m.group.Name, targets)
			m.cmdInput = newCmdPromptInput(m.width, m.height)
			m.cmdPrompt = true
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
				return m, nil
			}
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, &m.group, false, "")
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, &m.group, true, "")
		}
		if key.Matches(msg, m.keymap.ConnectSame) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectSameWindow(m.hostsToOpen(), m.opts, &m.group)
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			return m, func() tea.Msg { return openHostPickerMsg{groupIndex: m.groupIndex, returnTo: screenGroupHosts} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: m.groupIndex} }
		}
		if key.Matches(msg, m.keymap.HostConfig) && m.focus == focusList {
			return m, openHostConfigCmd(m.currentHost(), screenGroupHosts, &m.toast)
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			return m, copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenGroupHosts, &m.toast)
		}

	case toastMsg:
		m.toast = toast(msg)
		if m.opts.Popup && msg.level != toastErr {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
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
		return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, m.search.View(), right, placed, "")
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
	return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, m.search.View(), right, listContent, footer)
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
	return renderListEmptyState(m.width, m.height, m.search.Value(),
		dim.Render("No hosts in this group.")+"\n"+dim.Render("a \u2014 add hosts"))
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

func (m *groupHostsModel) IsQuitting() bool  { return m.quitting }
func (m *groupHostsModel) ExecCmd() []string { return m.execCmd }
