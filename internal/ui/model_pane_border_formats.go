package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type paneBorderFormatsCancelMsg struct{}

type paneBorderFormatsDoneMsg struct {
	value      string
	customFmts []string
}

type paneBorderFormatRow struct {
	label     string
	value     string
	deletable bool
}

func (r paneBorderFormatRow) Title() string       { return r.label }
func (r paneBorderFormatRow) Description() string { return "" }
func (r paneBorderFormatRow) FilterValue() string { return r.label }

type paneBorderFormatsDelegate struct{}

func (d paneBorderFormatsDelegate) Height() int                             { return 1 }
func (d paneBorderFormatsDelegate) Spacing() int                            { return 0 }
func (d paneBorderFormatsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d paneBorderFormatsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(paneBorderFormatRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	text := row.label
	if strings.TrimSpace(row.value) != "" {
		text = row.label + "  " + row.value
	}
	fmt.Fprint(w, renderSimpleRow(m.Width(), index == m.Index(), text, nil))
}

type paneBorderFormatsModel struct {
	parentCrumb string

	width  int
	height int

	includeInherit bool
	allowEdit      bool

	selected string
	custom   []string

	list list.Model

	adding     bool
	addInput   textinput.Model
	addToast   toast
	confirmDel bool
	delValue   string

	keymap keyMap
}

func (m *paneBorderFormatsModel) refreshAccentStyles() {
	if m.adding {
		setSearchFocused(&m.addInput, true)
	} else {
		setSearchFocused(&m.addInput, false)
	}
}

func newPaneBorderFormatsModel(defs config.Defaults, selected string, includeInherit bool, allowEdit bool) *paneBorderFormatsModel {
	selected = strings.TrimSpace(selected)
	if !includeInherit && selected == "" {
		selected = config.DefaultPaneBorderFormat
	}

	// Ensure the currently selected value is present in the list (for group overrides
	// that might reference a custom format not present in defaults).
	defsForRows := defs
	if selected != "" {
		defsForRows.PaneBorderFmt = selected
	}

	rows := buildPaneBorderFormatRows(defsForRows, includeInherit)
	items := make([]list.Item, 0, len(rows))
	selectedIdx := 0
	for i, r := range rows {
		items = append(items, r)
		if strings.TrimSpace(r.value) == selected {
			selectedIdx = i
		}
	}

	l := list.New(items, paneBorderFormatsDelegate{}, 0, 0)
	configureList(&l)
	l.SetDelegate(paneBorderFormatsDelegate{})
	if len(items) > 0 {
		l.Select(selectedIdx)
	}

	in := textinput.New()
	in.CharLimit = 2048
	in.Prompt = "new: "
	in.Placeholder = "tmux format string"
	configureSearch(&in)
	setSearchFocused(&in, true)

	return &paneBorderFormatsModel{
		includeInherit: includeInherit,
		allowEdit:      allowEdit,
		selected:       selected,
		custom:         append([]string(nil), defs.PaneBorderFmts...),
		list:           l,
		addInput:       in,
		keymap:         defaultKeyMap(),
	}
}

func buildPaneBorderFormatRows(defs config.Defaults, includeInherit bool) []paneBorderFormatRow {
	choices := paneBorderFormatChoices(defs)
	rows := make([]paneBorderFormatRow, 0, len(choices)+1)
	if includeInherit {
		rows = append(rows, paneBorderFormatRow{label: "inherit  (use defaults)", value: "", deletable: false})
	}
	// First is built-in default.
	if len(choices) > 0 {
		rows = append(rows, paneBorderFormatRow{label: "default", value: choices[0], deletable: false})
	}
	for i := 1; i < len(choices); i++ {
		rows = append(rows, paneBorderFormatRow{label: fmt.Sprintf("custom %d", i), value: choices[i], deletable: true})
	}
	return rows
}

func (m *paneBorderFormatsModel) Init() tea.Cmd { return nil }

func (m *paneBorderFormatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW, innerH := frameInnerSize(msg.Width, msg.Height)
		listH := innerH - 2 // header + footer
		if m.adding {
			listH -= 2 // input + spacer
		}
		m.list.SetSize(innerW, max(1, listH))
		m.addInput.Width = max(10, innerW-len(m.addInput.Prompt))
		return m, nil
	case tea.KeyMsg:
		if m.confirmDel {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				m.confirmDel = false
				m.deleteValue(m.delValue)
				m.delValue = ""
				return m, nil
			case "n", "N", "esc":
				m.confirmDel = false
				m.delValue = ""
				return m, nil
			default:
				return m, nil
			}
		}

		if m.adding {
			s := msg.String()
			switch s {
			case "esc":
				m.adding = false
				m.addInput.Blur()
				m.addInput.SetValue("")
				m.addToast = toast{}
				return m, nil
			case "enter":
				v := strings.TrimSpace(m.addInput.Value())
				m.addInput.SetValue("")
				if v == "" {
					m.addToast = toast{text: "format required", level: toastWarn}
					return m, nil
				}
				if !m.allowEdit {
					m.addToast = toast{text: "read-only", level: toastWarn}
					return m, nil
				}
				defs := config.Defaults{PaneBorderFmt: config.DefaultPaneBorderFormat, PaneBorderFmts: m.custom}
				if !addPaneBorderFormat(&defs, v) {
					m.addToast = toast{text: "already exists", level: toastWarn}
					return m, nil
				}
				m.custom = defs.PaneBorderFmts
				m.selected = v
				m.adding = false
				m.addInput.Blur()
				m.addToast = toast{text: "added", level: toastOK}
				m.rebuildList()
				return m, nil
			default:
				var cmd tea.Cmd
				m.addInput, cmd = m.addInput.Update(msg)
				return m, cmd
			}
		}

		if key.Matches(msg, m.keymap.Esc) {
			return m, func() tea.Msg { return paneBorderFormatsCancelMsg{} }
		}

		if msg.String() == "enter" {
			row, ok := m.list.SelectedItem().(paneBorderFormatRow)
			if !ok {
				return m, nil
			}
			val := strings.TrimSpace(row.value)
			if !m.includeInherit && val == "" {
				val = config.DefaultPaneBorderFormat
			}
			m.selected = val
			return m, func() tea.Msg {
				return paneBorderFormatsDoneMsg{value: m.selected, customFmts: append([]string(nil), m.custom...)}
			}
		}

		if m.allowEdit {
			s := msg.String()
			switch s {
			case "a":
				m.adding = true
				m.addToast = toast{}
				m.addInput.Focus()
				setSearchFocused(&m.addInput, true)
				// resize list for add mode
				if m.width > 0 && m.height > 0 {
					_, _ = m.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
				}
				return m, nil
			case "d":
				row, ok := m.list.SelectedItem().(paneBorderFormatRow)
				if !ok {
					return m, nil
				}
				if !row.deletable {
					m.addToast = toast{text: "can't delete default", level: toastWarn}
					return m, nil
				}
				m.confirmDel = true
				m.delValue = row.value
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *paneBorderFormatsModel) deleteValue(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return
	}
	if strings.TrimSpace(config.DefaultPaneBorderFormat) == v {
		m.addToast = toast{text: "can't delete default", level: toastWarn}
		return
	}

	defs := config.Defaults{PaneBorderFmt: config.DefaultPaneBorderFormat, PaneBorderFmts: m.custom}
	removePaneBorderFormat(&defs, v)
	m.custom = defs.PaneBorderFmts
	if strings.TrimSpace(m.selected) == v {
		m.selected = config.DefaultPaneBorderFormat
	}
	m.addToast = toast{text: "deleted", level: toastOK}
	m.rebuildList()
}

func (m *paneBorderFormatsModel) rebuildList() {
	defs := config.Defaults{PaneBorderFmt: m.selected, PaneBorderFmts: m.custom}
	rows := buildPaneBorderFormatRows(defs, m.includeInherit)
	items := make([]list.Item, 0, len(rows))
	idx := 0
	for i, r := range rows {
		items = append(items, r)
		if strings.TrimSpace(r.value) == strings.TrimSpace(m.selected) {
			idx = i
		}
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(idx)
	}
}

func (m *paneBorderFormatsModel) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var body strings.Builder
	if m.adding {
		body.WriteString(m.addInput.View())
		body.WriteString("\n\n")
	}
	body.WriteString(strings.TrimRight(m.list.View(), "\n"))

	headerRight := ""
	if !m.addToast.empty() {
		headerRight = renderToast(toast{text: truncateTail(m.addToast.text, 28), level: m.addToast.level})
	}

	footer := "⏎ select  Esc back"
	if m.confirmDel {
		footer = "y/⏎ delete  n/Esc cancel"
	} else if m.adding {
		footer = "⏎ add  Esc cancel"
	} else if m.allowEdit {
		footer = "⏎ select  a add  d delete  Esc back"
	}

	return renderFocusedFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Border formats"), headerRight, strings.TrimRight(body.String(), "\n"), styledFooter(footer))
}
