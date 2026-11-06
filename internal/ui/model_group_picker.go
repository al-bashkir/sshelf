package ui

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

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
	row := item.(groupPickRow)
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
		all = append(all, groupPickRow{index: i, name: config.GroupPath(g)})
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

// addingLines shows the hosts being added on at most two lines that fit
// innerW: "adding: a" or "adding N: a, b, ...", then names that do not fit
// as a dim "+N more" (or "+N") at the end. Returns nil when no hosts are
// set.
func (m *groupPickerModel) addingLines(innerW int) []string {
	n := len(m.hosts)
	if n == 0 {
		return nil
	}
	prefix := "adding: "
	if n > 1 {
		prefix = fmt.Sprintf("adding %d: ", n)
	}
	prefixW := lipgloss.Width(prefix)
	if innerW < prefixW+3 { // no room for names: the count only
		return []string{dim.Render(fmt.Sprintf("adding %d", n))}
	}

	p := hostLines{widths: [2]int{innerW - prefixW, innerW}}
	left := p.pack(m.hosts)
	out := []string{dim.Render(prefix) + strings.Join(p.lines[0], hostSep)}
	if len(p.lines[1]) > 0 {
		out = append(out, strings.Join(p.lines[1], hostSep))
	}
	if note := moreNote(p.used[len(out)-1], p.widths[len(out)-1], left); note != "" {
		out[len(out)-1] += dim.Render(note)
	}
	return out
}

const hostSep = ", "

// hostLines packs host names on two lines of fixed widths.
type hostLines struct {
	widths [2]int
	lines  [2][]string
	used   [2]int
}

// pack puts whole names on the first line, then on the second, and
// returns how many names did not fit.
func (p *hostLines) pack(hosts []string) (left int) {
	li := 0
	for i, h := range hosts {
		rest := len(hosts) - i - 1
		if p.add(li, h, rest) {
			continue
		}
		if li == 1 {
			return len(hosts) - i
		}
		li = 1
		p.add(li, h, rest) // an empty line takes any name
	}
	return 0
}

// add puts h on line li and reports true. A name that does not fit after
// others is not added (false); one alone on its line is cut to fit. While
// rest names follow, the second line keeps room for their moreNote.
func (p *hostLines) add(li int, h string, rest int) bool {
	sepW := 0
	if len(p.lines[li]) > 0 {
		sepW = len(hostSep)
	}
	w := lipgloss.Width(h)
	room := p.widths[li] - p.used[li] - sepW
	if li == 1 {
		room -= lipgloss.Width(moreNote(p.used[1]+sepW+w, p.widths[1], rest))
	}
	if w > room && sepW > 0 {
		return false
	}
	h = truncateTail(h, room) // room >= 3: addingLines shows names only from prefix+3 wide
	p.lines[li] = append(p.lines[li], h)
	p.used[li] += sepW + lipgloss.Width(h)
	return true
}

// moreNote counts n names left out: ", +N more" when it fits after usedW on
// a line width wide, else ", +N", and "" when n is 0.
func moreNote(usedW, width, n int) string {
	if n == 0 {
		return ""
	}
	if long := fmt.Sprintf(", +%d more", n); usedW+lipgloss.Width(long) <= width {
		return long
	}
	return fmt.Sprintf(", +%d", n)
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
