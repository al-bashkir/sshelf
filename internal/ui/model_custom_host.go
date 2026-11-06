package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type customHostModel struct {
	parentCrumb string

	returnTo   screen
	groupIndex int // -1: no group; Ctrl+A then asks for one

	width  int
	height int

	input  textinput.Model
	toast  toast
	keymap keyMap
}

func newCustomHostModel(groupIndex int, returnTo screen) *customHostModel {
	in := textinput.New()
	in.CharLimit = 512
	in.Width = 60
	in.Prompt = "host: "
	in.Placeholder = "host1 user@host2 ..."
	in.Focus()
	configureSearch(&in)
	setSearchFocused(&in, true)

	return &customHostModel{
		returnTo:   returnTo,
		groupIndex: groupIndex,
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
		switch {
		case key.Matches(msg, m.keymap.Esc):
			return m, msgCmd(customHostCancelMsg{})
		case key.Matches(msg, m.keymap.SelectAll):
			return m, m.submit(m.addMsg)
		case msg.String() == "enter":
			return m, m.submit(m.connectMsg)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// submit sends send(hosts) for the typed hosts; with none it warns.
func (m *customHostModel) submit(send func(hosts []string) tea.Msg) tea.Cmd {
	m.toast = toast{}
	hosts := strings.Fields(m.input.Value())
	if len(hosts) == 0 {
		m.toast = toast{text: "host required", level: toastWarn}
		return nil
	}
	return msgCmd(send(hosts))
}

func (m *customHostModel) connectMsg(hosts []string) tea.Msg {
	return customHostConnectMsg{returnTo: m.returnTo, groupIndex: m.groupIndex, hosts: hosts}
}

// addMsg adds hosts to the group, or asks for a group when none is set.
func (m *customHostModel) addMsg(hosts []string) tea.Msg {
	if m.groupIndex >= 0 {
		return customHostDoneMsg{returnTo: m.returnTo, groupIndex: m.groupIndex, hosts: hosts}
	}
	return customHostPickGroupMsg{returnTo: m.returnTo, hosts: hosts}
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

func (m *customHostModel) refreshAccentStyles() { setSearchFocused(&m.input, true) }
