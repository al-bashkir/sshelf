package ui

import (
	"fmt"

	"github.com/al-bashkir/sshelf/internal/sshcmd"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type hostPickerCancelMsg struct{}

type hostPickerDoneMsg struct {
	hosts []string
}

type hostPickerModel struct {
	hostSelectList
	pickerScreen
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
	m := &hostPickerModel{
		hostSelectList: newHostSelectList(all),
		pickerScreen:   newPickerScreen(opts, l),
	}
	m.bindList(&m.list)
	return m
}

func (m *hostPickerModel) Init() tea.Cmd { return nil }

func (m *hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.update(msg, pickerHooks{
		cancel:      hostPickerCancelMsg{},
		done:        func() tea.Cmd { return msgCmd(hostPickerDoneMsg{hosts: m.hostsToOpen()}) },
		applyFilter: m.applyFilter,
		more:        m.selectionActions(m.keymap),
	})
}

func (m *hostPickerModel) View() string {
	return m.view("Add hosts", "Add Hosts", m.helpKeys, m.statusLine())
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
