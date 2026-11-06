package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// formScreenCase is one form screen: how to open it, reach its User field,
// make its values unsavable and fix them again, and read its save message.
type formScreenCase struct {
	name          string
	new           func() tea.Model
	toUser        []string
	breakKeys     []string
	badValue      string
	fixKeys       []string
	savedUser     func(tea.Msg) (string, bool)
	toPicker      []string // reach a picker field; nil when the form has none
	cancelled     func(tea.Msg) bool
	borderPicker  func(tea.Model) *paneBorderFormatsModel // nil without a border_fmt field
	borderCrumb   string
	borderCurrent string
	savedBorder   func(tea.Msg) (string, []string)
}

func formScreenCases() []formScreenCase {
	defs := config.DefaultConfig().Defaults
	defs.PaneBorderFmts = []string{"#A"}
	return []formScreenCase{
		{
			name:   "host",
			new:    func() tea.Model { return newHostFormModel(2, config.Host{Host: "box"}, defs) },
			toUser: []string{"j"}, breakKeys: []string{"i", "backspace", "backspace", "backspace", "esc"},
			badValue: "host required", fixKeys: []string{"i", "b", "esc"},
			savedUser: func(msg tea.Msg) (string, bool) {
				s, ok := msg.(hostFormSaveMsg)
				return s.host.User, ok && s.index == 2
			},
			cancelled: func(msg tea.Msg) bool { _, ok := msg.(hostFormCancelMsg); return ok },
		},
		{
			name: "group",
			new: func() tea.Model {
				m := newGroupFormModel(1, config.Group{Name: "prod", PaneBorderFmt: "#G"}, nil, defs)
				m.parentCrumb = "Groups"
				return m
			},
			toUser: []string{"j", "j"}, breakKeys: []string{"i", "backspace", "backspace", "backspace", "backspace", "esc"}, // j j: past parent
			badValue: "name required", fixKeys: []string{"i", "p", "esc"},
			savedUser: func(msg tea.Msg) (string, bool) {
				s, ok := msg.(groupFormSaveMsg)
				return s.group.User, ok && s.index == 1
			},
			toPicker:     []string{"j", "j", "j", "j", "j", "j", "j"}, // open_mode
			cancelled:    func(msg tea.Msg) bool { _, ok := msg.(groupFormCancelMsg); return ok },
			borderPicker: func(m tea.Model) *paneBorderFormatsModel { return m.(*groupFormModel).borderPicker },
			borderCrumb:  "Groups", borderCurrent: "#G",
			savedBorder: func(msg tea.Msg) (string, []string) {
				g := msg.(groupFormSaveMsg).group
				return g.PaneBorderFmt, nil
			},
		},
		{
			name:   "settings",
			new:    func() tea.Model { return newDefaultsFormModel(defs, false) },
			toUser: nil, breakKeys: []string{"j", "i", "x", "esc"},
			badValue: "invalid port", fixKeys: []string{"i", "backspace", "esc"},
			savedUser: func(msg tea.Msg) (string, bool) {
				s, ok := msg.(defaultsFormSaveMsg)
				return s.defaults.User, ok
			},
			toPicker:     []string{"j", "j", "j", "j"}, // load_known_hosts
			cancelled:    func(msg tea.Msg) bool { _, ok := msg.(defaultsFormCancelMsg); return ok },
			borderPicker: func(m tea.Model) *paneBorderFormatsModel { return m.(*defaultsFormModel).borderPicker },
			borderCrumb:  "Settings", borderCurrent: config.DefaultPaneBorderFormat,
			savedBorder: func(msg tea.Msg) (string, []string) {
				d := msg.(defaultsFormSaveMsg).defaults
				return d.PaneBorderFmt, d.PaneBorderFmts
			},
		},
	}
}

func plainView(m tea.Model) string { return ansi.Strip(m.View()) }

func TestFormScreensSaveWithCtrlS(t *testing.T) {
	for _, c := range formScreenCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			keys(m, c.toUser...)
			keys(m, "i", "ops", "esc")
			if user, ok := c.savedUser(keys(m, "ctrl+s")); !ok || user != "ops" {
				t.Fatalf("Ctrl+S saved user %q (ok=%v), want ops", user, ok)
			}
		})
	}
}

// The footer offers Ctrl+S in insert mode: it ends insert mode and saves
// what was typed. The typed key's command is a cursor blink, so it does
// not run.
func TestFormScreensSaveFromInsertMode(t *testing.T) {
	for _, c := range formScreenCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			keys(m, c.toUser...)
			m.Update(hk("i"))
			m.Update(hk("ops"))
			if user, ok := c.savedUser(keys(m, "ctrl+s")); !ok || user != "ops" {
				t.Fatalf("Ctrl+S in insert mode saved user %q (ok=%v), want ops", user, ok)
			}
			if strings.Contains(plainView(m), "INSERT") {
				t.Fatal("still in insert mode after Ctrl+S")
			}
		})
	}
}

func TestFormScreensKeepBadValues(t *testing.T) {
	for _, c := range formScreenCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			keys(m, c.breakKeys...)
			if got := keys(m, "ctrl+s"); got != nil {
				t.Fatalf("Ctrl+S with a bad value emitted %T", got)
			}
			if !strings.Contains(plainView(m), c.badValue) {
				t.Fatalf("view does not show %q:\n%s", c.badValue, plainView(m))
			}
			keys(m, c.fixKeys...)
			if _, ok := c.savedUser(keys(m, "ctrl+s")); !ok {
				t.Fatal("Ctrl+S after the fix did not save")
			}
			if strings.Contains(plainView(m), c.badValue) {
				t.Fatalf("the old error still shows after a save")
			}
		})
	}
}

func TestFormScreensFitTheWindow(t *testing.T) {
	for _, c := range formScreenCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
			lines := strings.Split(m.View(), "\n")
			if len(lines) != 30 {
				t.Fatalf("%d lines, want 30", len(lines))
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w != 90 {
					t.Fatalf("line %d is %d wide, want 90", i, w)
				}
			}
		})
	}
}

func TestFormScreensPickerTakesScreenKeys(t *testing.T) {
	for _, c := range formScreenCases() {
		if c.toPicker == nil {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			keys(m, c.toPicker...)
			for _, k := range []string{"enter", "q", "alt+1", "alt+2", "ctrl+s", "esc"} {
				if got := keys(m, k); got != nil {
					t.Fatalf("%s with the picker open emitted %T", k, got)
				}
			}
			if !c.cancelled(keys(m, "esc")) {
				t.Fatal("Esc after the picker closed did not cancel the form")
			}
		})
	}
}

func TestFormScreensBorderFormatSubModal(t *testing.T) {
	for _, c := range formScreenCases() {
		if c.borderPicker == nil {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			pw, ph := pickerModalSize(100, 40)
			for _, open := range []string{"enter", "space", "l"} {
				m := c.new()
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
				keys(m, "k") // border_fmt is the last field
				if got := keys(m, open); got != nil {
					t.Fatalf("%s emitted %T", open, got)
				}
				p := c.borderPicker(m)
				if p == nil {
					t.Fatalf("%s did not open the border formats", open)
				}
				if p.parentCrumb != c.borderCrumb || p.selected != c.borderCurrent {
					t.Fatalf("crumb %q selected %q, want %q and %q", p.parentCrumb, p.selected, c.borderCrumb, c.borderCurrent)
				}
				if p.width != pw || p.height != ph {
					t.Fatalf("sub-modal %dx%d, want %dx%d", p.width, p.height, pw, ph)
				}
				if !strings.Contains(plainView(m), "Border format") {
					t.Fatalf("view does not show the sub-modal:\n%s", plainView(m))
				}
			}

			m := c.new()
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			keys(m, "k", "enter")
			// The sub-modal takes every key, Esc too.
			if _, ok := keys(m, "esc").(paneBorderFormatsCancelMsg); !ok {
				t.Fatal("Esc did not go to the sub-modal")
			}
			m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
			if w, h := pickerModalSize(120, 50); c.borderPicker(m).width != w || c.borderPicker(m).height != h {
				t.Fatal("resize did not reach the sub-modal")
			}
			m.Update(paneBorderFormatsCancelMsg{})
			if c.borderPicker(m) != nil {
				t.Fatal("cancel did not close the sub-modal")
			}

			keys(m, "enter")
			m.Update(paneBorderFormatsDoneMsg{value: " #X ", customFmts: []string{"#A", "#X"}})
			if c.borderPicker(m) != nil {
				t.Fatal("done did not close the sub-modal")
			}
			keys(m, "enter")
			if got := c.borderPicker(m).selected; got != "#X" {
				t.Fatalf("reopened on %q, want the chosen #X", got)
			}
			m.Update(paneBorderFormatsCancelMsg{})
			got, custom := c.savedBorder(keys(m, "ctrl+s"))
			if got != "#X" {
				t.Fatalf("saved border format %q, want #X", got)
			}
			if c.name == "settings" && !slices.Equal(custom, []string{"#A", "#X"}) {
				t.Fatalf("saved custom formats %q, want #A #X", custom)
			}
		})
	}
}

func TestSettingsQuitQuestion(t *testing.T) {
	defs := config.DefaultConfig().Defaults
	open := func() *defaultsFormModel {
		m := newDefaultsFormModel(defs, true)
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		return m
	}
	asking := func(m *defaultsFormModel) bool { return m.View() == renderQuitConfirm(100, 40) }

	m := open()
	if got := keys(m, "q"); got != nil || !asking(m) || m.toast.text != "quit? (y/n)" {
		t.Fatalf("q: emitted %T asking=%v toast %q", got, asking(m), m.toast.text)
	}
	if got := keys(m, "x"); got != nil || !asking(m) {
		t.Fatal("other keys must leave the question open")
	}
	for _, no := range []string{"n", "N", "esc"} {
		keys(m, "q")
		if got := keys(m, no); got != nil || asking(m) || !m.toast.empty() {
			t.Fatalf("%s: emitted %T asking=%v toast %q", no, got, asking(m), m.toast.text)
		}
	}
	for _, yes := range []string{"y", "Y", "enter"} {
		m := open()
		keys(m, "q")
		if _, ok := keys(m, yes).(tea.QuitMsg); !ok {
			t.Fatalf("%s did not quit", yes)
		}
	}
	// The quit question wins over the unsaved-changes one.
	m = open()
	dirty(t, m)
	keys(m, "q")
	if !asking(m) {
		t.Fatal("dirty settings with confirm on did not ask to quit")
	}
	if _, ok := keys(m, "y").(tea.QuitMsg); !ok {
		t.Fatal("y did not quit")
	}
}

func TestSettingsCleanTabSwitch(t *testing.T) {
	for k, to := range map[string]screen{"alt+1": screenHosts, "alt+2": screenGroups} {
		m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
		if s, ok := keys(m, k).(switchScreenMsg); !ok || s.to != to {
			t.Fatalf("%s emitted %#v, want switch to %v", k, s, to)
		}
	}
}

// The keys typed here return a cursor blink, so the test checks state: a
// key the screen took would not be typed, and on this dirty form a tab or
// quit key would open the unsaved-changes question.
func TestSettingsInsertModeTypesScreenKeys(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	for _, k := range []string{"i", "x", "q", "alt+1", "alt+2"} {
		m.Update(hk(k))
	}
	if !m.form.editing || m.form.askingDiscard() || !strings.HasPrefix(m.form.value("user"), "xq") {
		t.Fatalf("editing=%v asking=%v user %q, want still typing xq", m.form.editing, m.form.askingDiscard(), m.form.value("user"))
	}
}

func TestSettingsViewShowsPicker(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	keys(m, "j", "j", "j", "j", "enter") // load_known_hosts
	if !strings.Contains(plainView(m), "▸ yes") {
		t.Fatalf("picker popup missing:\n%s", plainView(m))
	}
}

// Settings keeps its toast and footer in the window, when the form is
// taller than the window and when it is shorter.
func TestSettingsViewKeepsToastAndFooter(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	check := func(want ...string) {
		t.Helper()
		for _, h := range []int{20, 60} {
			m.Update(tea.WindowSizeMsg{Width: 100, Height: h})
			for _, w := range want {
				if v := plainView(m); !strings.Contains(v, w) {
					t.Fatalf("height %d: %q missing:\n%s", h, w, v)
				}
			}
		}
	}
	check("Ctrl+S save")
	keys(m, "j", "i", "x", "esc", "ctrl+s") // bad port
	check("invalid port", "Ctrl+S save")
}

// A picker low in a scrolled form opens next to its field.
func TestSettingsViewShowsPickerInScrolledForm(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	keys(m, "k", "k", "enter") // border_pos, the last picker
	if !strings.Contains(plainView(m), "▸ bottom") {
		t.Fatalf("picker popup missing:\n%s", plainView(m))
	}
}

// With no room for content, Settings still shows the focused field's row.
func TestSettingsViewKeepsFocusedRowInATinyWindow(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 7}) // one content row
	if !strings.Contains(plainView(m), "User") {
		t.Fatalf("focused field missing:\n%s", plainView(m))
	}
}

func TestSettingsFieldHelpPreview(t *testing.T) {
	const helper = "Show helper text below focused form fields"
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	keys(m, "j", "j", "j", "j", "j", "j", "j") // show_field_help
	if !strings.Contains(plainView(m), helper) {
		t.Fatalf("help text missing:\n%s", plainView(m))
	}
	keys(m, "enter", "j", "enter") // yes -> no
	if strings.Contains(plainView(m), helper) {
		t.Fatal("help text still shows after choosing no")
	}
}

// Group: the border formats offer "inherit" and cannot be changed.
// Settings: no "inherit", and a can add a format.
func TestFormScreensBorderFormatChoices(t *testing.T) {
	for _, c := range formScreenCases() {
		if c.borderPicker == nil {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			keys(m, "k", "enter")
			group := c.name == "group"
			if got := strings.Contains(plainView(m), "inherit  (use defaults)"); got != group {
				t.Fatalf("inherit row shown=%v, want %v:\n%s", got, group, plainView(m))
			}
			m.Update(hk("a"))
			if got := strings.Contains(plainView(m), "new: "); got == group {
				t.Fatalf("a opened the add box=%v, want %v:\n%s", got, !group, plainView(m))
			}
		})
	}
}

func TestSettingsPortAndThresholdAreNarrow(t *testing.T) {
	m := newDefaultsFormModel(config.DefaultConfig().Defaults, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if p, th, u := m.form.input("port").Width, m.form.input("threshold").Width, m.form.input("user").Width; p != 12 || th != 12 || u != 77 {
		t.Fatalf("port %d, threshold %d, user %d wide, want 12, 12 and 77", p, th, u)
	}
}

func TestSettingsColorschemeOverridesTheAccent(t *testing.T) {
	for cs, overridden := range map[string]bool{"nord": true, "default": false, "": false} {
		d := config.DefaultConfig().Defaults
		d.Colorscheme = cs
		m := newDefaultsFormModel(d, false)
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		if got := strings.Contains(plainView(m), "overridden by nord theme"); got != overridden {
			t.Errorf("colorscheme %q: accent overridden %v, want %v", cs, got, overridden)
		}
	}
}
