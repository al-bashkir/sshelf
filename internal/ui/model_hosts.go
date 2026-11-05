package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/al-bashkir/sshelf/internal/hosts"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type focusState int

const (
	focusList focusState = iota
	focusSearch
)

type hostRow struct {
	host           string
	selected       bool
	hasCfg         bool
	hidden         bool
	matchedIndexes []int
}

func (i hostRow) Title() string       { return i.host }
func (i hostRow) Description() string { return "" }
func (i hostRow) FilterValue() string { return i.host }

type hostDelegate struct{}

func (d hostDelegate) Height() int                             { return 1 }
func (d hostDelegate) Spacing() int                            { return 0 }
func (d hostDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d hostDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(hostRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderHostLikeRow(m.Width(), index == m.Index(), row.selected, row.host, row.hasCfg, row.hidden, row.matchedIndexes))
}

type knownHostsReloadMsg struct {
	res  hosts.LoadResult
	errs []hosts.PathError
}

type hostsModel struct {
	hostSelectList
	connectActions
	opts Options

	width  int
	height int

	keymap keyMap

	list   list.Model
	search textinput.Model
	focus  focusState

	reloading   bool
	showHidden  bool
	prevSearch  string
	navPendingG bool
	confirmQuit bool

	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model
}

func newHostsModel(opts Options) *hostsModel {
	l := list.New(nil, hostDelegate{}, 0, 0) // items come from applyFilter below
	l.Title = "Hosts"
	configureList(&l)

	search := newSearchInput()

	m := &hostsModel{
		hostSelectList: newHostSelectList(opts.Hosts),
		opts:           opts,
		keymap:         defaultKeyMap(),
		list:           l,
		search:         search,
		focus:          focusList,
	}
	m.bindList(&m.list)
	if !opts.Config.Defaults.LoadKnownHosts {
		m.keymap.Reload.SetEnabled(false)
	}
	m.applyFilter("")
	return m
}

func (m *hostsModel) Init() tea.Cmd {
	return nil
}

func (m *hostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case knownHostsReloadMsg:
		m.reloaded(msg)
		return m, nil
	case tea.KeyMsg:
		if cmd, ok := m.handleKey(msg); ok {
			return m, cmd
		}
	case toastMsg:
		return m, m.showToast(toast(msg))
	}
	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *hostsModel) layout() {
	innerW := max(0, m.width-2)
	m.list.SetSize(innerW, tabBoxListContentHeight(m.width, m.height))
	const countsW = 24 // room for the counts right of the search box
	m.search.Width = max(10, innerW-countsW-len(m.search.Prompt))
}

func (m *hostsModel) reloaded(msg knownHostsReloadMsg) {
	spinnerStop()
	m.reloading = false
	m.opts.SkippedLines = msg.res.SkippedLines
	m.opts.LoadErrors = msg.errs
	m.setHosts(msg.res.Hosts)
	m.toast = toast{text: fmt.Sprintf("%d hosts loaded", len(m.allHosts)), level: toastInfo}
}

// showToast shows the result of a finished command. In popup mode anything
// but an error ends sshelf.
func (m *hostsModel) showToast(t toast) tea.Cmd {
	m.toast = t
	if m.opts.Popup && t.level != toastErr {
		m.quitting = true
		return tea.Quit
	}
	return nil
}

// handleKey gives a key to the open overlay first, then to the focus keys,
// then to the screen actions. ok is false when search or list should get it.
func (m *hostsModel) handleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if cmd, ok := m.overlayKey(msg); ok {
		return cmd, true
	}
	if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
		return nil, true
	}
	return runKeyAction(msg, m.focus, m.keyActions())
}

// overlayKey handles keys while help, the command prompt or a confirm
// question shows, and vim g/gg/G in the list.
func (m *hostsModel) overlayKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case m.showHelp:
		m.helpKey(msg)
		return nil, true
	case m.cmdPrompt:
		return handleCmdPromptKey(msg, &m.cmdPrompt, &m.cmdInput, &m.toast, m.handleConnectWithRemoteCommand), true
	}
	if handled, cmd := handleConfirmQuit(msg, &m.confirmQuit, &m.toast, &m.quitting, true); handled {
		return cmd, true
	}
	if handled, cmd := handleConfirmConnect(msg, &m.confirmConnect, &m.pendingConnectFn, &m.toast); handled {
		return cmd, true
	}
	return nil, m.focus == focusList && handleVimListNav(msg, &m.list, &m.navPendingG)
}

func (m *hostsModel) helpKey(msg tea.KeyMsg) {
	if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
		m.showHelp = false
		return
	}
	updateHelpViewport(&m.helpVP, msg)
}

func (m *hostsModel) keyActions() []keyAction {
	km := m.keymap
	return []keyAction{
		{keys: km.GroupsTab, run: func() tea.Cmd { return msgCmd(switchScreenMsg{to: screenGroups}) }},
		{keys: km.Settings, run: func() tea.Cmd { return msgCmd(openDefaultsFormMsg{returnTo: screenHosts}) }},
		{keys: km.Quit, inList: true, run: m.quit},
		{keys: km.Help, run: m.toggleHelp},
		{keys: km.Esc, run: func() tea.Cmd {
			// Esc priority: blur search → clear selection → clear search.
			handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter)
			return nil
		}},
		{keys: km.Reload, inList: true, run: m.reload},
		{keys: km.ToggleSel, inList: true, run: func() tea.Cmd { m.toggleCurrentSelection(); return nil }},
		{keys: km.SelectAll, inList: true, run: func() tea.Cmd { m.selectAllFiltered(); return nil }},
		{keys: km.ClearSel, inList: true, run: func() tea.Cmd { m.clearSelection(); return nil }},
		{keys: km.ConnectCmd, inList: true, run: m.openCmdPrompt},
		{keys: km.Connect, run: m.enter},
		{keys: km.OneWindow, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connect(m.hostsToOpen(), m.opts, nil, true, "")
		}},
		{keys: km.ConnectSame, inList: true, run: func() tea.Cmd {
			m.toast = toast{}
			return m.connectSameWindow(m.hostsToOpen(), m.opts, nil)
		}},
		{keys: km.AddHosts, inList: true, run: m.addToGroup},
		{keys: km.CustomHost, inList: true, run: func() tea.Cmd {
			return msgCmd(openCustomHostMsg{returnTo: screenHosts, groupIndex: -1})
		}},
		{keys: km.HostConfig, inList: true, run: func() tea.Cmd { return openHostConfigCmd(m.currentHost(), screenHosts, &m.toast) }},
		{keys: km.Copy, inList: true, run: func() tea.Cmd {
			return copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenHosts, &m.toast)
		}},
		{keys: km.HideHost, inList: true, run: m.toggleCurrentHidden},
		{keys: km.ShowHidden, inList: true, run: func() tea.Cmd {
			m.showHidden = !m.showHidden
			m.applyFilter(m.search.Value())
			return nil
		}},
	}
}

func (m *hostsModel) quit() tea.Cmd {
	if !m.opts.Config.Defaults.ConfirmQuit {
		m.quitting = true
		return tea.Quit
	}
	m.confirmQuit = true
	m.toast = toast{text: "quit? (y/n)", level: toastWarn}
	return nil
}

func (m *hostsModel) toggleHelp() tea.Cmd {
	m.showHelp = !m.showHelp
	if m.showHelp && m.width > 0 && m.height > 0 {
		m.helpVP = initHelpViewport(m.width, m.height, "Hosts", m.helpKeys())
	}
	return nil
}

// reload re-reads known_hosts. The key is bound only while known_hosts
// loading is on; the app keeps the binding in step with the setting.
func (m *hostsModel) reload() tea.Cmd {
	m.toast = toast{text: "reloading", level: toastInfo}
	m.reloading = true
	spinnerStart()
	return tea.Batch(
		reloadKnownHostsCmd(m.opts.KnownHosts),
		tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} }),
	)
}

func (m *hostsModel) openCmdPrompt() tea.Cmd {
	targets := m.hostsToOpen()
	if len(targets) == 0 {
		return nil
	}
	m.cmdPromptCrumb = cmdPromptCrumb("Hosts", targets)
	m.cmdInput = newCmdPromptInput(m.width, m.height)
	m.cmdPrompt = true
	return nil
}

// enter accepts the query in the search box, or connects from the list.
func (m *hostsModel) enter() tea.Cmd {
	if m.focus == focusSearch {
		acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
		return nil
	}
	m.toast = toast{}
	return m.connect(m.hostsToOpen(), m.opts, nil, false, "")
}

func (m *hostsModel) addToGroup() tea.Cmd {
	hostsToAdd := m.hostsToOpen()
	if len(hostsToAdd) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	return msgCmd(openGroupPickerMsg{hosts: hostsToAdd})
}

// setHosts replaces the host list, drops selections of hosts that are gone,
// and reapplies the current filter.
func (m *hostsModel) setHosts(hs []string) {
	m.opts.Hosts = hs
	m.allHosts = append([]string(nil), hs...)
	present := make(map[string]struct{}, len(m.allHosts))
	for _, h := range m.allHosts {
		present[h] = struct{}{}
	}
	for h := range m.selected {
		if _, ok := present[h]; !ok {
			delete(m.selected, h)
		}
	}
	m.applyFilter(m.search.Value())
}

func (m *hostsModel) applyFilter(query string) {
	m.filter(query, m.opts.Inventory, !m.showHidden, true)
}

type toastMsg toast

func (m *hostsModel) toggleCurrentHidden() tea.Cmd {
	host := m.currentHost()
	if host == "" {
		return nil
	}
	hide := !isHostHidden(m.opts.Inventory, host)
	return func() tea.Msg { return toggleHiddenHostMsg{host: host, hide: hide} }
}

func (m *hostsModel) reapplyFilter() {
	m.applyFilter(m.search.Value())
}

func (m *hostsModel) hiddenCount() int { return m.hiddenN }

func (m *hostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	if strings.TrimSpace(remoteCmd) == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}
	return m.connect(m.hostsToOpen(), m.opts, nil, false, remoteCmd)
}

func (m *hostsModel) helpKeys() helpMap {
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
				m.keymap.GroupsTab,
				m.keymap.Settings,
				m.keymap.Esc,
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
				m.keymap.CustomHost,
				m.keymap.HostConfig,
				m.keymap.AddHosts,
				m.keymap.Copy,
				m.keymap.HideHost,
				m.keymap.ShowHidden,
			}},
			{title: "General", keys: []key.Binding{
				m.keymap.Settings,
				m.keymap.Reload,
				m.keymap.Help,
				m.keymap.Quit,
			}},
		},
	}
}

func reloadKnownHostsCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		res, errs := hosts.LoadKnownHosts(paths)
		return knownHostsReloadMsg{res: res, errs: errs}
	}
}
