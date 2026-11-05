package ui

import (
	"fmt"
	"io"
	"slices"
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
	setSearchFocused(&m.addInput, m.adding)
}

func newPaneBorderFormatsModel(defs config.Defaults, selected string, includeInherit bool, allowEdit bool) *paneBorderFormatsModel {
	selected = strings.TrimSpace(selected)
	if !includeInherit && selected == "" {
		selected = config.DefaultPaneBorderFormat
	}

	l := list.New(nil, paneBorderFormatsDelegate{}, 0, 0)
	configureList(&l)
	l.SetDelegate(paneBorderFormatsDelegate{})

	in := textinput.New()
	in.CharLimit = 2048
	in.Prompt = "new: "
	in.Placeholder = "tmux format string"
	configureSearch(&in)

	m := &paneBorderFormatsModel{
		includeInherit: includeInherit,
		allowEdit:      allowEdit,
		selected:       selected,
		custom:         append([]string(nil), defs.PaneBorderFmts...),
		list:           l,
		addInput:       in,
		keymap:         defaultKeyMap(),
	}

	// Ensure the currently selected value is present in the list (for group overrides
	// that might reference a custom format not present in defaults).
	if selected != "" {
		defs.PaneBorderFmt = selected
	}
	m.setRows(buildPaneBorderFormatRows(defs, includeInherit))
	return m
}

func buildPaneBorderFormatRows(defs config.Defaults, includeInherit bool) []paneBorderFormatRow {
	var rows []paneBorderFormatRow
	if includeInherit {
		rows = append(rows, paneBorderFormatRow{label: "inherit  (use defaults)"})
	}
	for i, v := range paneBorderFormatChoices(defs) {
		row := paneBorderFormatRow{label: fmt.Sprintf("custom %d", i), value: v, deletable: true}
		if i == 0 { // the built-in default comes first
			row = paneBorderFormatRow{label: "default", value: v}
		}
		rows = append(rows, row)
	}
	return rows
}

func (m *paneBorderFormatsModel) Init() tea.Cmd { return nil }

func (m *paneBorderFormatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case tea.KeyMsg:
		if m.confirmDel {
			m.confirmDeleteKey(msg.String())
			return m, nil
		}
		if m.adding {
			return m, m.addKey(msg)
		}
		if cmd, ok := m.browseKey(msg); ok {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// layout fits the list and the add input into the frame. While the input
// shows, it takes two list rows.
func (m *paneBorderFormatsModel) layout() {
	innerW, innerH := frameInnerSize(m.width, m.height)
	listH := innerH - 2 // header + footer
	if m.adding {
		listH -= 2 // input + spacer
	}
	m.list.SetSize(innerW, max(1, listH))
	m.addInput.Width = max(10, innerW-len(m.addInput.Prompt))
}

// browseKey handles a key in list mode. ok is false when the list itself
// should get the key (cursor moves).
func (m *paneBorderFormatsModel) browseKey(msg tea.KeyMsg) (cmd tea.Cmd, ok bool) {
	switch {
	case key.Matches(msg, m.keymap.Esc):
		return func() tea.Msg { return paneBorderFormatsCancelMsg{} }, true
	case msg.String() == "enter":
		return m.choose(), true
	case !m.allowEdit:
		return nil, false
	case msg.String() == "a":
		m.addToast = toast{}
		m.setAdding(true)
	case msg.String() == "d":
		m.askDelete()
	default:
		return nil, false
	}
	return nil, true
}

// choose sends the selected format and the custom list to the parent.
func (m *paneBorderFormatsModel) choose() tea.Cmd {
	row, _ := m.list.SelectedItem().(paneBorderFormatRow) // the default row is always there
	m.selected = strings.TrimSpace(row.value)
	done := paneBorderFormatsDoneMsg{value: m.selected, customFmts: append([]string(nil), m.custom...)}
	return func() tea.Msg { return done }
}

func (m *paneBorderFormatsModel) askDelete() {
	row, _ := m.list.SelectedItem().(paneBorderFormatRow)
	if !row.deletable {
		m.addToast = toast{text: "can't delete default", level: toastWarn}
		return
	}
	m.confirmDel = true
	m.delValue = row.value
}

// confirmDeleteKey answers the delete prompt; other keys keep it open.
func (m *paneBorderFormatsModel) confirmDeleteKey(k string) {
	switch k {
	case "y", "Y", "enter":
		m.deleteValue(m.delValue)
	case "n", "N", "esc":
	default:
		return
	}
	m.confirmDel = false
	m.delValue = ""
}

// addKey handles a key while the add input shows.
func (m *paneBorderFormatsModel) addKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.addInput.SetValue("")
		m.addToast = toast{}
		m.setAdding(false)
	case "enter":
		m.submitAdd()
	default:
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return cmd
	}
	return nil
}

func (m *paneBorderFormatsModel) submitAdd() {
	v := strings.TrimSpace(m.addInput.Value())
	m.addInput.SetValue("")
	if v == "" {
		m.addToast = toast{text: "format required", level: toastWarn}
		return
	}
	defs := config.Defaults{PaneBorderFmts: m.custom}
	if !addPaneBorderFormat(&defs, v) {
		m.addToast = toast{text: "already exists", level: toastWarn}
		return
	}
	m.custom = defs.PaneBorderFmts
	m.selected = v
	m.addToast = toast{text: "added", level: toastOK}
	m.setAdding(false)
	m.rebuildList()
}

// setAdding shows or hides the add input and resizes the list to match.
func (m *paneBorderFormatsModel) setAdding(on bool) {
	m.adding = on
	if on {
		m.addInput.Focus()
	} else {
		m.addInput.Blur()
	}
	m.refreshAccentStyles()
	m.layout()
}

// deleteValue removes custom format v; askDelete only offers custom rows.
func (m *paneBorderFormatsModel) deleteValue(v string) {
	defs := config.Defaults{PaneBorderFmts: m.custom}
	removePaneBorderFormat(&defs, v)
	m.custom = defs.PaneBorderFmts
	if m.selected == v {
		m.selected = config.DefaultPaneBorderFormat
	}
	m.addToast = toast{text: "deleted", level: toastOK}
	m.rebuildList()
}

func (m *paneBorderFormatsModel) rebuildList() {
	defs := config.Defaults{PaneBorderFmt: m.selected, PaneBorderFmts: m.custom}
	m.setRows(buildPaneBorderFormatRows(defs, m.includeInherit))
}

// setRows fills the list and puts the cursor on the selected value.
func (m *paneBorderFormatsModel) setRows(rows []paneBorderFormatRow) {
	items := make([]list.Item, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	m.list.SetItems(items)
	sel := strings.TrimSpace(m.selected)
	m.list.Select(max(0, slices.IndexFunc(rows, func(r paneBorderFormatRow) bool { return strings.TrimSpace(r.value) == sel })))
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
