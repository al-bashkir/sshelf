package ui

import (
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// flabForm is one form screen, opened with a value in every field, and the
// schema of the fields it shows.
type flabForm struct {
	name   string
	new    func() tea.Model
	schema formSchema
}

func flabForms() []flabForm {
	defs := config.DefaultConfig().Defaults
	return []flabForm{
		{"settings", func() tea.Model { return newDefaultsFormModel(defs, false) }, settingsSchema()},
		{"group", func() tea.Model { return newGroupFormModel(1, config.Group{Name: "prod", User: "admin"}, defs) }, groupSchema(defs)},
		{"host", func() tea.Model { return newHostFormModel(2, config.Host{Host: "box", User: "admin"}, defs) }, hostSchema(defs)},
	}
}

// flabOpen opens the form in a 100x40 window.
func flabOpen(f flabForm) tea.Model {
	m := f.new()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	return m
}

// flabRow finds the field row of label in view: the box border, a
// two-column mark ("  " or "> "), then "label:". It returns the row and the
// text after the colon.
func flabRow(view, label string) (row int, rest string, ok bool) {
	for i, line := range strings.Split(view, "\n") {
		r := []rune(line)
		if len(r) < 3 || r[0] != '│' {
			continue
		}
		if after, found := strings.CutPrefix(string(r[3:]), label+":"); found {
			return i, after, true
		}
	}
	return 0, "", false
}

// flabValueCol is the screen column where the value of label's field
// starts, or -1 when view has no such row or the row has no value.
func flabValueCol(view, label string) int {
	row, rest, ok := flabRow(view, label)
	if !ok {
		return -1
	}
	value := strings.TrimLeft(rest, " ")
	if value == "" || strings.HasPrefix(value, "│") {
		return -1
	}
	line := strings.Split(view, "\n")[row]
	return lipgloss.Width(line[:len(line)-len(value)])
}

// flabFocused is the label of the row that shows the focus mark "> ".
func flabFocused(view string) string {
	for _, line := range strings.Split(view, "\n") {
		if label, _, ok := strings.Cut(strings.TrimPrefix(line, "│> "), ":"); ok && strings.HasPrefix(line, "│> ") {
			return label
		}
	}
	return ""
}

// Every value in a form starts in the same column, whatever the length of
// its label.
func TestFormValuesStartInOneColumn(t *testing.T) {
	for _, f := range flabForms() {
		t.Run(f.name, func(t *testing.T) {
			view := plainView(flabOpen(f))
			first, firstCol := "", -1
			for _, sec := range f.schema.Sections {
				for _, fd := range sec.Fields {
					col := flabValueCol(view, fd.Label)
					if col < 0 {
						t.Fatalf("no value shown for %q:\n%s", fd.Label, view)
					}
					if firstCol < 0 {
						first, firstCol = fd.Label, col
					}
					if col != firstCol {
						t.Errorf("%q value starts at column %d, %q value at %d", fd.Label, col, first, firstCol)
					}
				}
			}
			if t.Failed() {
				t.Log(view)
			}
		})
	}
}

// An open picker popup starts in the column where its field's value starts.
func TestFormPickerPopupsLineUpWithTheirValue(t *testing.T) {
	for _, f := range flabForms() {
		t.Run(f.name, func(t *testing.T) {
			pickers := map[string]bool{}
			for _, sec := range f.schema.Sections {
				for _, fd := range sec.Fields {
					if fd.Kind == fieldPicker {
						pickers[fd.Label] = true
					}
				}
			}
			m := flabOpen(f)
			checked := 0
			for seen := map[string]bool{}; ; sendKeys(m, "j") {
				view := plainView(m)
				label := flabFocused(view)
				if label == "" || seen[label] {
					break
				}
				seen[label] = true
				if !pickers[label] {
					continue
				}
				valueCol := flabValueCol(view, label)
				sendKeys(m, "enter")
				open := plainView(m)
				popupCol := -1
				for _, line := range strings.Split(open, "\n") {
					if i := strings.Index(line, "┌ "+label+" "); i >= 0 {
						popupCol = lipgloss.Width(line[:i])
					}
				}
				if popupCol < 0 || popupCol != valueCol {
					t.Errorf("%q popup starts at column %d, its value at %d:\n%s", label, popupCol, valueCol, open)
				}
				sendKeys(m, "esc")
				checked++
			}
			if checked != len(pickers) {
				t.Fatalf("checked %d pickers, the form has %d", checked, len(pickers))
			}
		})
	}
}
