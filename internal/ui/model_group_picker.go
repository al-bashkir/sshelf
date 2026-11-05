package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sahilm/fuzzy"
)

type groupPickRow struct {
	index          int
	name           string
	matchedIndexes []int
}

func (i groupPickRow) Title() string       { return i.name }
func (i groupPickRow) Description() string { return "" }
func (i groupPickRow) FilterValue() string { return i.name }

type groupPickerDelegate struct{}

func (d groupPickerDelegate) Height() int                             { return 1 }
func (d groupPickerDelegate) Spacing() int                            { return 0 }
func (d groupPickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupPickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(groupPickRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderSimpleRow(m.Width(), index == m.Index(), row.name, row.matchedIndexes))
}

type groupPickerModel struct {
	pickerScreen

	hosts []string // the hosts being added, shown above the search (topLines)

	all   []groupPickRow
	names []string // cached for fuzzy search, parallel to all
}

func newGroupPickerModel(opts Options) *groupPickerModel {
	all := make([]groupPickRow, 0, len(opts.Inventory.Groups))
	names := make([]string, 0, len(opts.Inventory.Groups))
	items := make([]list.Item, 0, len(opts.Inventory.Groups))
	for i, g := range opts.Inventory.Groups {
		all = append(all, groupPickRow{index: i, name: g.Name})
	}
	sort.SliceStable(all, func(i, j int) bool {
		return strings.ToLower(all[i].name) < strings.ToLower(all[j].name)
	})
	for _, row := range all {
		names = append(names, row.name)
		items = append(items, row)
	}

	l := list.New(items, groupPickerDelegate{}, 0, 0)
	return &groupPickerModel{
		pickerScreen: newPickerScreen(opts, l),
		all:          all,
		names:        names,
	}
}

func (m *groupPickerModel) Init() tea.Cmd { return nil }

func (m *groupPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		innerW, _ := frameInnerSize(size.Width, size.Height)
		m.topLines = m.addingLines(innerW)
	}
	return m, m.update(msg, pickerHooks{cancel: groupPickerCancelMsg{}, done: m.done, applyFilter: m.applyFilter})
}

// done picks the current group; with no group shown it does nothing.
func (m *groupPickerModel) done() tea.Cmd {
	row, ok := m.list.SelectedItem().(groupPickRow)
	if !ok {
		return nil
	}
	return msgCmd(groupPickerDoneMsg{groupIndex: row.index})
}

// addingLines packs the pending host names into at most 2 lines, each fitting
// within innerW. The first line is prefixed with "adding: " or "adding N: ".
// If hosts overflow 2 lines, the last line ends with a dim "+N more" count.
// Returns nil when no hosts are set.
func (m *groupPickerModel) addingLines(innerW int) []string {
	if len(m.hosts) == 0 {
		return nil
	}

	prefix := "adding: "
	if len(m.hosts) > 1 {
		prefix = fmt.Sprintf("adding %d: ", len(m.hosts))
	}
	prefixW := lipgloss.Width(prefix)

	// Narrow-width fallback: just show count, no individual names.
	if innerW < prefixW+3 {
		return []string{dim.Render(fmt.Sprintf("adding %d", len(m.hosts)))}
	}

	// Two-line budgets: line 0 is narrowed by the prefix, line 1 is full width.
	budgets := [2]int{max(1, innerW-prefixW), max(1, innerW)}

	type lineBuf struct {
		hosts []string
		usedW int
	}
	bufs := [2]lineBuf{}
	li := 0

	for i, h := range m.hosts {
		sep := ""
		if len(bufs[li].hosts) > 0 {
			sep = ", "
		}
		sepW := lipgloss.Width(sep)

		// Truncate hostname if it would exceed the remaining budget.
		avail := budgets[li] - bufs[li].usedW - sepW
		if avail < 1 {
			avail = 1
		}
		h = truncateTail(h, avail)
		tokenW := sepW + lipgloss.Width(h)

		// On the last line, reserve room for a "+N more" trailer.
		isLastBudget := li == len(budgets)-1
		remaining := len(m.hosts) - i - 1
		trailerW := 0
		if isLastBudget && remaining > 0 {
			trailerW = lipgloss.Width(fmt.Sprintf(", +%d more", remaining))
			// If the long trailer doesn't fit, try a shorter form.
			if bufs[li].usedW+tokenW+trailerW > budgets[li] {
				trailerW = lipgloss.Width(fmt.Sprintf(", +%d", remaining))
			}
		}

		if bufs[li].usedW+tokenW+trailerW <= budgets[li] || len(bufs[li].hosts) == 0 {
			bufs[li].hosts = append(bufs[li].hosts, h)
			bufs[li].usedW += tokenW
		} else if li < len(budgets)-1 {
			// Wrap to the next line.
			li++
			bufs[li].hosts = append(bufs[li].hosts, h)
			bufs[li].usedW = lipgloss.Width(h)
		} else {
			// Last line is full — remaining hosts become the "+N more" count.
			break
		}
	}

	placed := len(bufs[0].hosts) + len(bufs[1].hosts)
	leftOver := len(m.hosts) - placed

	var out []string
	for i, buf := range bufs {
		if len(buf.hosts) == 0 {
			continue
		}
		var sb strings.Builder
		if i == 0 {
			sb.WriteString(dim.Render(prefix))
		}
		sb.WriteString(strings.Join(buf.hosts, ", "))
		if i == li && leftOver > 0 {
			sb.WriteString(dim.Render(fmt.Sprintf(", +%d more", leftOver)))
		}
		out = append(out, sb.String())
	}
	return out
}

func (m *groupPickerModel) View() string {
	return m.view("Select group", "Select Group", m.helpKeys, m.statusLine())
}

func (m *groupPickerModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	selectGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
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
				selectGroup,
			}},
			{title: "General", keys: []key.Binding{
				esc,
				m.keymap.Help,
			}},
		},
	}
}

func (m *groupPickerModel) statusLine() string {
	left := fmt.Sprintf("groups: %d/%d", len(m.list.Items()), len(m.all))
	return pickerStatusLine(left, &m.list, m.search.Value(), m.toast)
}

// applyFilter shows every group for an empty query, else the fuzzy
// matches with their letters marked, and moves to the first row.
func (m *groupPickerModel) applyFilter(query string) {
	items := make([]list.Item, 0, len(m.all))
	if query = strings.TrimSpace(query); query == "" {
		for _, r := range m.all { // rows in all carry no match marks
			items = append(items, r)
		}
	} else {
		for _, mt := range fuzzy.Find(query, m.names) {
			r := m.all[mt.Index]
			r.matchedIndexes = mt.MatchedIndexes
			items = append(items, r)
		}
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}
