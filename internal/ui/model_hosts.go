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

	case knownHostsReloadMsg:
		spinnerStop()
		m.reloading = false
		m.opts.SkippedLines = msg.res.SkippedLines
		m.opts.LoadErrors = msg.errs
		m.setHosts(msg.res.Hosts)
		m.toast = toast{text: fmt.Sprintf("%d hosts loaded", len(m.allHosts)), level: toastInfo}
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

		if handled, cmd := handleConfirmQuit(msg, &m.confirmQuit, &m.toast, &m.quitting, true); handled {
			return m, cmd
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

		if key.Matches(msg, m.keymap.GroupsTab) {
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
		}
		if key.Matches(msg, m.keymap.Settings) {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenHosts} }
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
				m.helpVP = initHelpViewport(m.width, m.height, "Hosts", m.helpKeys())
			}
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			// Esc priority: blur search → clear selection → clear search.
			handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter)
			return m, nil
		}

		if key.Matches(msg, m.keymap.Reload) && m.focus != focusSearch {
			if !m.opts.Config.Defaults.LoadKnownHosts {
				m.toast = toast{text: "known_hosts disabled", level: toastWarn}
				return m, nil
			}
			m.toast = toast{text: "reloading", level: toastInfo}
			m.reloading = true
			spinnerStart()
			return m, tea.Batch(
				reloadKnownHostsCmd(m.opts.KnownHosts),
				tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} }),
			)
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
			m.cmdPromptCrumb = cmdPromptCrumb("Hosts", targets)
			m.cmdInput = newCmdPromptInput(m.width, m.height)
			m.cmdPrompt = true
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			// In search focus, Enter should accept the query and go back to list.
			if m.focus == focusSearch {
				acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
				return m, nil
			}
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, nil, false, "")
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, nil, true, "")
		}
		if key.Matches(msg, m.keymap.ConnectSame) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectSameWindow(m.hostsToOpen(), m.opts, nil)
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			hostsToAdd := m.hostsToOpen()
			if len(hostsToAdd) == 0 {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			return m, func() tea.Msg { return openGroupPickerMsg{hosts: hostsToAdd} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenHosts, groupIndex: -1} }
		}
		if key.Matches(msg, m.keymap.HostConfig) && m.focus == focusList {
			return m, openHostConfigCmd(m.currentHost(), screenHosts, &m.toast)
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			return m, copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenHosts, &m.toast)
		}
		if key.Matches(msg, m.keymap.HideHost) && m.focus == focusList {
			return m, m.toggleCurrentHidden()
		}
		if key.Matches(msg, m.keymap.ShowHidden) && m.focus == focusList {
			m.showHidden = !m.showHidden
			m.applyFilter(m.search.Value())
			return m, nil
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
