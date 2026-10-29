package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/sshelf/internal/sshcmd"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type hostPickerCancelMsg struct{}

type hostPickerDoneMsg struct {
	hosts []string
}

type hostPickerModel struct {
	hostSelectList
	opts        Options
	parentCrumb string

	width  int
	height int

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap   keyMap
	showHelp bool
	toast    toast

	prevSearch string
}

func newHostPickerModel(opts Options) *hostPickerModel {
	all := append([]string(nil), opts.Hosts...)
	items := make([]list.Item, 0, len(all))
	for _, h := range all {
		_, ok := sshcmd.FindHostConfig(opts.Inventory.Hosts, h)
		items = append(items, hostRow{host: h, hasCfg: ok})
	}

	l := list.New(items, hostDelegate{}, 0, 0)
	l.Title = "Add hosts"
	configureList(&l)

	search := newSearchInput()

	m := &hostPickerModel{
		hostSelectList: newHostSelectList(all),
		opts:           opts,
		list:           l,
		search:         search,
		focus:          focusList,
		keymap:         defaultKeyMap(),
	}
	m.bindList(&m.list)
	return m
}

func (m *hostPickerModel) Init() tea.Cmd { return nil }

func (m *hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW, innerH := frameInnerSize(w, h)
		m.list.SetSize(innerW, max(1, innerH-5))
		m.search.Width = max(10, innerW-len(m.search.Prompt))
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			if handlePickerEsc(&m.focus, &m.search, &m.prevSearch, m.applyFilter) {
				return m, nil
			}
			return m, func() tea.Msg { return hostPickerCancelMsg{} }
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
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
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				setFocus(&m.focus, &m.search, focusList)
				return m, nil
			}
			picked := m.hostsToOpen()
			return m, func() tea.Msg { return hostPickerDoneMsg{hosts: picked} }
		}
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *hostPickerModel) View() string {
	if m.showHelp {
		return renderHelpModal(m.width, m.height, "Add Hosts", m.helpKeys())
	}
	innerW, _ := frameInnerSize(m.width, m.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	searchLine := m.search.View()
	listView := strings.TrimRight(m.list.View(), "\n")
	body := strings.TrimRight(searchLine+"\n"+sep+"\n"+listView+"\n"+sep, "\n")
	return renderFocusedFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Add hosts"), "", body, m.statusLine())
}

func (m *hostPickerModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	add := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "add"),
	)

	return helpMap{
		sections: []helpSection{
			{title: "Navigation", keys: []key.Binding{
				m.list.KeyMap.CursorUp,
				m.list.KeyMap.CursorDown,
				m.list.KeyMap.PrevPage,
				m.list.KeyMap.NextPage,
				m.keymap.FocusSearch,
				m.keymap.ToggleFocus,
			}},
			{title: "Selection", keys: []key.Binding{
				m.keymap.ToggleSel,
				m.keymap.SelectAll,
				m.keymap.ClearSel,
				add,
			}},
			{title: "General", keys: []key.Binding{
				esc,
				m.keymap.Help,
			}},
		},
	}
}

func (m *hostPickerModel) statusLine() string {
	left := fmt.Sprintf("hosts: %d/%d  sel:%d", len(m.list.Items()), len(m.allHosts), len(m.selected))
	return pickerStatusLine(left, &m.list, m.search.Value(), m.toast)
}

func (m *hostPickerModel) applyFilter(query string) {
	m.filter(query, m.opts.Inventory, false, false)
}
