package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sahilm/fuzzy"
)

type groupsDelegate struct{}

func (d groupsDelegate) Height() int                             { return 1 }
func (d groupsDelegate) Spacing() int                            { return 0 }
func (d groupsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(groupRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
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
	connectActions
	opts Options

	width  int
	height int

	allRows []groupRow
	rows    []groupRow

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap         keyMap
	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model

	confirmQuit   bool
	confirmDelete bool
	deleteIndex   int

	prevSearch  string
	navPendingG bool
}

func newGroupsModel(opts Options) *groupsModel {
	rows := groupsRows(opts.Inventory)
	items := make([]list.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, r)
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetDelegate(groupsDelegate{})
	l.Title = "Groups"
	configureList(&l)

	search := newSearchInput()

	m := &groupsModel{
		opts:     opts,
		allRows:  rows,
		rows:     rows,
		list:     l,
		search:   search,
		focus:    focusList,
		keymap:   defaultKeyMap(),
		showHelp: false,
	}
	return m
}

func groupsRows(inv config.Inventory) []groupRow {
	rows := make([]groupRow, 0, len(inv.Groups))
	for i, g := range inv.Groups {
		rows = append(rows, groupRow{index: i, name: g.Name, hostCount: len(g.Hosts)})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
	})
	return rows
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
	m.allRows = groupsRows(inv)
	m.applyFilter(m.search.Value())
}

func (m *groupsModel) Init() tea.Cmd { return nil }

func (m *groupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW := max(0, w-2)
		m.list.SetSize(innerW, tabBoxListContentHeight(w, h))
		promptW := len(m.search.Prompt)
		reserve := 18
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
			run := func(cmd string) tea.Cmd { return m.connectAllCmd(false, cmd) }
			return m, handleCmdPromptKey(msg, &m.cmdPrompt, &m.cmdInput, &m.toast, run)
		}

		if handled, cmd := handleConfirmQuit(msg, &m.confirmQuit, &m.toast, &m.quitting, true); handled {
			return m, cmd
		}
		if m.confirmDelete {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				idx := m.deleteIndex
				m.confirmDelete = false
				return m, func() tea.Msg { return deleteGroupMsg{index: idx} }
			case "n", "N", "esc":
				m.confirmDelete = false
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
		if key.Matches(msg, m.keymap.Settings) {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenGroups} }
		}

		if key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch {
			if !m.opts.Config.Defaults.ConfirmQuit {
				m.quitting = true
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp && m.width > 0 && m.height > 0 {
				m.helpVP = initHelpViewport(m.width, m.height, "Groups", m.helpKeys())
			}
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			handleEscChain(&m.focus, &m.search, &m.prevSearch, 0, nil, m.applyFilter)
			return m, nil
		}
		if key.Matches(msg, m.keymap.NewGroup) && m.focus == focusList {
			return m, func() tea.Msg { return openGroupFormMsg{index: -1} }
		}
		if key.Matches(msg, m.keymap.EditGroup) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openGroupFormMsg{index: row.index} }
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openHostPickerMsg{groupIndex: row.index, returnTo: screenGroups} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenGroups, groupIndex: row.index} }
		}
		if key.Matches(msg, m.keymap.DeleteGroup) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			m.confirmDelete = true
			m.deleteIndex = row.index
			m.toast = toast{text: "delete? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
				return m, nil
			}
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openGroupHostsMsg{index: row.index} }
		}
		if key.Matches(msg, m.keymap.ConnectCmd) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			m.cmdPromptCrumb = "Groups > " + row.name
			m.cmdInput = newCmdPromptInput(m.width, m.height)
			m.cmdPrompt = true
			return m, nil
		}
		if key.Matches(msg, m.keymap.ConnectAll) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectAllCmd(false, "")
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok || row.index < 0 || row.index >= len(m.opts.Inventory.Groups) {
				m.toast = toast{text: "no group selected", level: toastWarn}
				return m, nil
			}
			g := m.opts.Inventory.Groups[row.index]
			g.Name = suggestCopyGroupName(m.opts.Inventory, g.Name)
			return m, func() tea.Msg { return openGroupFormPrefillMsg{group: g} }
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectAllCmd(true, "")
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

func (m *groupsModel) View() string {
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Groups", m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		return renderCmdPromptModal(m.width, m.height, m.cmdPromptCrumb,
			"Connect and run a remote command for all hosts (keeps sessions open).", m.cmdInput)
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.confirmConnect {
		modal := connectConfirmBox(max(0, m.width-4), m.confirmConnectCount, m.confirmConnectHosts)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}
	if m.confirmDelete {
		name := ""
		hostCount := 0
		if m.deleteIndex >= 0 && m.deleteIndex < len(m.opts.Inventory.Groups) {
			name = m.opts.Inventory.Groups[m.deleteIndex].Name
			hostCount = len(m.opts.Inventory.Groups[m.deleteIndex].Hosts)
		}
		innerW := max(0, m.width-2)
		innerH := max(0, m.height-2)
		contentH := max(0, innerH-4)
		modal := deleteGroupConfirmBox(innerW, name, hostCount)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		right := statusDot(true, false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
		return renderMainTabBox(m.width, m.height, 1, m.search.View(), right, placed)
	}

	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(true, false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
	}
	var footer string
	if m.width < compactWidthThreshold {
		footer = styledFooter("\u21b5 open  C connect  ? help")
	} else {
		footer = styledFooter("\u21b5 open  C connect  ·  o panes  Ctrl+o cmd  ·  n new")
		if m.height >= twoLineFooterMinHeight {
			footer += "\n" + styledFooter("e edit  d delete  y copy  a add hosts  c custom  ·  Alt+1 hosts  Alt+3 settings  tab search  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	return renderMainTabBoxWithFooter(m.width, m.height, 1, m.search.View(), right, listContent, footer)
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

func (m *groupsModel) emptyStateView() string {
	return renderListEmptyState(m.width, m.height, m.search.Value(),
		dim.Render("No groups yet.")+"\n"+dim.Render("n \u2014 create a new group"))
}

func (m *groupsModel) applyFilter(query string) {
	var prevName string
	prevIndex := -1
	if row, ok := m.list.SelectedItem().(groupRow); ok {
		prevName = row.name
		prevIndex = row.index
	}

	query = strings.TrimSpace(query)
	var rows []groupRow
	if query == "" {
		rows = append([]groupRow(nil), m.allRows...)
	} else {
		matches := fuzzy.Find(query, groupNames(m.allRows))
		rows = make([]groupRow, 0, len(matches))
		for _, mt := range matches {
			r := m.allRows[mt.Index]
			r.matchedIndexes = mt.MatchedIndexes
			rows = append(rows, r)
		}
	}
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
	if row.index < 0 || row.index >= len(m.opts.Inventory.Groups) {
		m.toast = toast{text: "invalid group", level: toastErr}
		return nil
	}
	g := m.opts.Inventory.Groups[row.index]
	if len(g.Hosts) == 0 {
		m.toast = toast{text: "group has no hosts", level: toastWarn}
		return nil
	}
	return m.connect(append([]string(nil), g.Hosts...), m.opts, &g, oneWindow, remoteCmd)
}

func (m *groupsModel) IsQuitting() bool  { return m.quitting }
func (m *groupsModel) ExecCmd() []string { return m.execCmd }
