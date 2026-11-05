package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/al-bashkir/sshelf/internal/config"
)

// borderModel opens the border-format picker over custom formats #A and #B
// in an 80x24 terminal.
func borderModel(inherit, edit bool, selected string) *paneBorderFormatsModel {
	m := newPaneBorderFormatsModel(config.Defaults{PaneBorderFmts: []string{"#A", "#B"}}, selected, inherit, edit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

// keys sends each key in turn (see hk) and returns the message the last
// key's command produces, if any. Only the last command runs: a key typed
// into a text input returns a cursor blink that sleeps.
func keys(m tea.Model, ks ...string) tea.Msg {
	var cmd tea.Cmd
	for _, k := range ks {
		_, cmd = m.Update(hk(k))
	}
	if cmd == nil {
		return nil
	}
	return cmd()
}

func chosen(t *testing.T, m *paneBorderFormatsModel) paneBorderFormatsDoneMsg {
	t.Helper()
	done, ok := keys(m, "enter").(paneBorderFormatsDoneMsg)
	if !ok {
		t.Fatal("enter did not choose a format")
	}
	return done
}

func viewHas(t *testing.T, m *paneBorderFormatsModel, want ...string) {
	t.Helper()
	v := m.View()
	for _, w := range want {
		if !strings.Contains(v, w) {
			t.Fatalf("view lacks %q:\n%s", w, v)
		}
	}
}

func viewLacks(t *testing.T, m *paneBorderFormatsModel, s string) {
	t.Helper()
	if v := m.View(); strings.Contains(v, s) {
		t.Fatalf("view has %q:\n%s", s, v)
	}
}

func TestBorderFormatsChoose(t *testing.T) {
	tc := []struct {
		name     string
		inherit  bool
		selected string
		want     string
	}{
		{"custom", false, "#B", "#B"},
		{"trimmed", false, " #A ", "#A"},
		{"inherit row", true, "", ""},
		{"no inherit: blank means default", false, "", config.DefaultPaneBorderFormat},
		{"unknown value is listed", true, "#X", "#X"},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			got := chosen(t, borderModel(c.inherit, true, c.selected))
			want := paneBorderFormatsDoneMsg{value: c.want, customFmts: []string{"#A", "#B"}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v want %+v", got, want)
			}
		})
	}
}

func TestBorderFormatsEscCancels(t *testing.T) {
	if _, ok := keys(borderModel(false, true, ""), "esc").(paneBorderFormatsCancelMsg); !ok {
		t.Fatal("esc did not cancel")
	}
}

func TestBorderFormatsAdd(t *testing.T) {
	m := borderModel(false, true, "#A")
	keys(m, "a")
	viewHas(t, m, "new: ", "⏎ add  Esc cancel")

	keys(m, "#C", "enter")
	viewHas(t, m, "added", "a add  d delete")
	viewLacks(t, m, "new: ")

	got := chosen(t, m)
	if want := (paneBorderFormatsDoneMsg{value: "#C", customFmts: []string{"#A", "#B", "#C"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestBorderFormatsAddRejects(t *testing.T) {
	m := borderModel(false, true, "#A")
	keys(m, "a", "enter")
	viewHas(t, m, "format required", "⏎ add  Esc cancel")

	keys(m, " #A ", "enter")
	viewHas(t, m, "already exists", "⏎ add  Esc cancel")

	keys(m, config.DefaultPaneBorderFormat, "enter")
	viewHas(t, m, "already exists")

	keys(m, "#typed", "esc")
	viewHas(t, m, "a add  d delete")
	viewLacks(t, m, "already exists")

	keys(m, "a") // the input starts empty again
	if v := m.addInput.Value(); v != "" {
		t.Fatalf("input %q after reopen, want empty", v)
	}
	keys(m, "esc")
	if got := chosen(t, m); !reflect.DeepEqual(got.customFmts, []string{"#A", "#B"}) {
		t.Fatalf("custom formats %q changed", got.customFmts)
	}
}

func TestBorderFormatsDelete(t *testing.T) {
	m := borderModel(false, true, "#A")
	keys(m, "d")
	viewHas(t, m, "y/⏎ delete  n/Esc cancel")

	keys(m, "x") // other keys wait for an answer
	viewHas(t, m, "y/⏎ delete  n/Esc cancel")

	keys(m, "n")
	viewHas(t, m, "a add  d delete")

	keys(m, "d", "esc")
	viewHas(t, m, "a add  d delete")

	keys(m, "d", "y")
	viewHas(t, m, "deleted")
	got := chosen(t, m) // the deleted format was selected: back to default
	if want := (paneBorderFormatsDoneMsg{value: config.DefaultPaneBorderFormat, customFmts: []string{"#B"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestBorderFormatsDeleteWithEnterKeepsOtherSelection(t *testing.T) {
	m := borderModel(false, true, "#B")
	keys(m, "up", "d", "enter") // #B selected; delete #A, the row above it
	got := chosen(t, m)
	if want := (paneBorderFormatsDoneMsg{value: "#B", customFmts: []string{"#B"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

// The add input takes two list rows; leaving add mode gives them back.
func TestBorderFormatsAddModeListHeight(t *testing.T) {
	m := borderModel(false, true, "#A")
	full := m.list.Height()

	keys(m, "a")
	if h := m.list.Height(); h != full-2 {
		t.Fatalf("adding: list height %d, want %d", h, full-2)
	}
	keys(m, "esc")
	if h := m.list.Height(); h != full {
		t.Fatalf("after esc: list height %d, want %d", h, full)
	}
	keys(m, "a", "#C", "enter")
	if h := m.list.Height(); h != full {
		t.Fatalf("after add: list height %d, want %d", h, full)
	}
}

func TestBorderFormatsCannotDeleteDefault(t *testing.T) {
	for name, m := range map[string]*paneBorderFormatsModel{
		"default row": borderModel(false, true, ""),
		"inherit row": borderModel(true, true, ""),
	} {
		keys(m, "d")
		if v := m.View(); !strings.Contains(v, "can't delete default") || !strings.Contains(v, "a add  d delete") {
			t.Fatalf("%s: d did not refuse:\n%s", name, v)
		}
	}
}

// Rows show label and value, one per line, with the cursor on the selection.
func TestBorderFormatsRows(t *testing.T) {
	lines := strings.Split(borderModel(true, true, "#A").View(), "\n")
	at := func(s string) int {
		for i, l := range lines {
			if strings.Contains(l, s) {
				return i
			}
		}
		t.Fatalf("no line has %q:\n%s", s, strings.Join(lines, "\n"))
		return -1
	}
	inherit, def, a, b := at("inherit  (use defaults)"), at("default  #[bg=green"), at("▸ custom 1  #A"), at("  custom 2  #B")
	if def != inherit+1 || a != def+1 || b != a+1 {
		t.Fatalf("rows not on adjacent lines: %d %d %d %d", inherit, def, a, b)
	}
	if strings.Contains(lines[inherit], "▸") {
		t.Fatal("cursor on the inherit row, want custom 1")
	}
}

func TestBorderFormatsLayout(t *testing.T) {
	m := borderModel(false, true, "#A")
	if n := len(strings.Split(m.View(), "\n")); n != 24 {
		t.Fatalf("view is %d lines, want 24", n)
	}

	keys(m, "a", strings.Repeat("x", 200)) // long input scrolls inside the frame
	for _, l := range strings.Split(m.View(), "\n") {
		if w := lipgloss.Width(l); w > 80 {
			t.Fatalf("line %d wide, want <= 80: %q", w, l)
		}
	}

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 3}) // tiny: keep one row
	if h := m.list.Height(); h != 1 {
		t.Fatalf("tiny terminal: list height %d, want 1", h)
	}
}

func TestBorderFormatsLongInputShowsItsEnd(t *testing.T) {
	m := borderModel(false, true, "#A")
	keys(m, "a", strings.Repeat("x", 200)+"END")
	viewHas(t, m, "END")
}

func TestBorderFormatsArrowsMoveInReadOnlyMode(t *testing.T) {
	m := borderModel(false, false, "#A")
	keys(m, "down")
	if got := chosen(t, m); got.value != "#B" {
		t.Fatalf("chose %q after down, want #B", got.value)
	}
}

func TestBorderFormatsViewNeedsSize(t *testing.T) {
	m := newPaneBorderFormatsModel(config.Defaults{}, "", false, true)
	if v := m.View(); v != "" {
		t.Fatalf("no size: view %q, want empty", v)
	}
	for _, s := range []tea.WindowSizeMsg{{Width: 80, Height: 0}, {Width: 0, Height: 24}} {
		m.Update(s)
		if v := m.View(); v != "" {
			t.Fatalf("%dx%d: view %q, want empty", s.Width, s.Height, v)
		}
	}
	for _, s := range []tea.WindowSizeMsg{{Width: 1, Height: 24}, {Width: 80, Height: 1}} {
		m.Update(s)
		if v := m.View(); v == "" {
			t.Fatalf("%dx%d: empty view, want a frame", s.Width, s.Height)
		}
	}
}

func TestBorderFormatsReadOnly(t *testing.T) {
	m := borderModel(false, false, "#A")
	keys(m, "a")
	viewLacks(t, m, "new: ")
	keys(m, "d")
	viewHas(t, m, "⏎ select  Esc back")
	if got := chosen(t, m); got.value != "#A" {
		t.Fatalf("chose %q, want #A", got.value)
	}
}
