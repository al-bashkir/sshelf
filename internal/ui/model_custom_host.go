package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type customHostModel struct {
	opts        Options
	parentCrumb string

	returnTo   screen
	groupIndex int
	groupName  string

	width  int
	height int

	input  textinput.Model
	toast  toast
	keymap keyMap
}

func newCustomHostModel(opts Options, groupIndex int, returnTo screen) *customHostModel {
	name := ""
	if groupIndex >= 0 && groupIndex < len(opts.Inventory.Groups) {
		name = strings.TrimSpace(opts.Inventory.Groups[groupIndex].Name)
	}

	in := textinput.New()
	in.CharLimit = 512
	in.Width = 60
	in.Prompt = "host: "
	in.Placeholder = "host1 user@host2 ..."
	in.Focus()
	configureSearch(&in)
	setSearchFocused(&in, true)

	return &customHostModel{
		opts:       opts,
		returnTo:   returnTo,
		groupIndex: groupIndex,
		groupName:  name,
		input:      in,
		keymap:     defaultKeyMap(),
	}
}

func (m *customHostModel) Init() tea.Cmd { return nil }

func (m *customHostModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW, _ := frameInnerSize(msg.Width, msg.Height)
		m.input.Width = max(10, innerW-len(m.input.Prompt))
		return m, nil
	case tea.KeyMsg:
		if key.Matches(msg, m.keymap.Esc) {
			return m, func() tea.Msg { return customHostCancelMsg{} }
		}
		if key.Matches(msg, m.keymap.SelectAll) {
			m.toast = toast{}
			hosts := strings.Fields(strings.TrimSpace(m.input.Value()))
			if len(hosts) == 0 {
				m.toast = toast{text: "host required", level: toastWarn}
				return m, nil
			}
			if m.groupIndex >= 0 {
				return m, func() tea.Msg { return customHostDoneMsg{returnTo: m.returnTo, groupIndex: m.groupIndex, hosts: hosts} }
			}
			return m, func() tea.Msg { return customHostPickGroupMsg{returnTo: m.returnTo, hosts: hosts} }
		}
		if msg.String() == "enter" {
			m.toast = toast{}
			hosts := strings.Fields(strings.TrimSpace(m.input.Value()))
			if len(hosts) == 0 {
				m.toast = toast{text: "host required", level: toastWarn}
				return m, nil
			}
			return m, func() tea.Msg {
				return customHostConnectMsg{returnTo: m.returnTo, groupIndex: m.groupIndex, hosts: hosts}
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *customHostModel) View() string {
	title := breadcrumbTitle(m.parentCrumb, "Connect")

	var b strings.Builder
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(styledFooter("⏎ connect  Ctrl+a add to group  Esc cancel"))

	footer := ""
	if !m.toast.empty() {
		footer = renderToast(m.toast)
	}

	return renderFocusedFrame(m.width, m.height, title, "", strings.TrimRight(b.String(), "\n"), footer)
}
