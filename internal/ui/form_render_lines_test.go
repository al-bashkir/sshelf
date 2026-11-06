package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// frlRows is the plain view of m as rows, without the box sides and
// trailing spaces.
func frlRows(m tea.Model) []string {
	rows := strings.Split(plainView(m), "\n")
	for i, r := range rows {
		r = strings.TrimSuffix(strings.TrimPrefix(r, "│"), "│")
		rows[i] = strings.TrimRight(r, " ")
	}
	return rows
}

// frlShows fails unless want is a run of consecutive rows in m's view.
func frlShows(t *testing.T, m tea.Model, want ...string) {
	t.Helper()
	rows := frlRows(m)
	for i := 0; i+len(want) <= len(rows); i++ {
		if slices.Equal(rows[i:i+len(want)], want) {
			return
		}
	}
	t.Fatalf("view lacks the rows\n%s\nin\n%s", strings.Join(want, "\n"), strings.Join(rows, "\n"))
}

// frlSection is the plain section header row of the given width.
func frlSection(label string, w int) string {
	return strings.TrimRight(ansi.Strip(formSection(label, w)), " ")
}

// frlSettings opens Settings in a 100x40 window (98 columns inside the box)
// and sends ks.
func frlSettings(edit func(*config.Defaults), ks ...string) tea.Model {
	d := config.DefaultConfig().Defaults
	if edit != nil {
		edit(&d)
	}
	m := newDefaultsFormModel(d, false)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	sendKeys(m, ks...)
	return m
}

func frlHelpOff(d *config.Defaults) { d.ShowFieldHelp = false }

// frlGroupForm opens the group form on g in a 100x40 window and sends ks.
func frlGroupForm(g config.Group, ks ...string) tea.Model {
	m := newGroupFormModel(1, g, config.DefaultConfig().Defaults)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	sendKeys(m, ks...)
	return m
}

// frlLines is the form content of f at innerW as plain rows.
func frlLines(f *formModel, innerW int) ([]string, int) {
	lines, focusLine := f.renderFormContent(innerW)
	for i, l := range lines {
		lines[i] = ansi.Strip(l)
	}
	return lines, focusLine
}

// Each field kind on Settings: text with its placeholder, narrow number,
// pickers with ▾, and the sub-modal value cut at the box edge. Only the
// focused field gets the > mark and its help line.
func TestSettingsRowsShowEachFieldKind(t *testing.T) {
	m := frlSettings(nil)
	frlShows(t, m,
		frlSection("SSH", 98),
		"> User:            login username",
		"  ╰ Default SSH username for all connections",
		"  Port:            22",
		"  Identity file:   ~/.ssh/id_ed25519",
		"  Extra arguments: -o Option=value ...")
	frlShows(t, m,
		frlSection("Appearance", 98),
		"  Colorscheme:     default ▾",
		"  Accent color:    default ▾")
	frlShows(t, m,
		frlSection("Tmux", 98),
		"  Tmux mode:       auto ▾",
		"  Open mode:       auto ▾",
		"  Session name:    sshelf",
		frlSection("Behavior", 98))
	frlShows(t, m,
		"  Border position: bottom ▾",
		("  Border format:   " + config.DefaultPaneBorderFormat)[:98])
}

// The help line follows the focus, and field help off hides it.
func TestSettingsHelpLineFollowsFocus(t *testing.T) {
	frlShows(t, frlSettings(nil, "j"),
		"  User:            login username",
		"> Port:            22",
		"  ╰ Default SSH port (1–65535)",
		"  Identity file:   ~/.ssh/id_ed25519")
	frlShows(t, frlSettings(frlHelpOff, "j"),
		"  User:            login username",
		"> Port:            22",
		"  Identity file:   ~/.ssh/id_ed25519")
}

// A bad value shows its error in place of the help line, also with field
// help off, and only while its field has focus.
func TestSettingsErrorShowsUnderTheFocusedField(t *testing.T) {
	const bad = "  ╰ must be a number 1-65535"
	frlShows(t, frlSettings(frlHelpOff, "j", "i", "x", "esc"), "> Port:            22x", bad, "  Identity file:   ~/.ssh/id_ed25519")

	m := frlSettings(nil, "j", "i", "x", "esc")
	frlShows(t, m, "> Port:            22x", bad, "  Identity file:   ~/.ssh/id_ed25519")
	sendKeys(m, "j")
	frlShows(t, m, "• Port:            22x", "> Identity file:   ~/.ssh/id_ed25519", "  ╰ Path to SSH private key for authentication")
	sendKeys(m, "k", "i", "backspace", "esc")
	frlShows(t, m, "> Port:            22", "  ╰ Default SSH port (1–65535)")
}

// A changed field shows • once focus leaves it; unchanged fields keep the
// plain indent.
func TestSettingsMarksChangedFields(t *testing.T) {
	m := frlSettings(nil, "i", "x", "esc")
	frlShows(t, m, "> User:            x", "  ╰ Default SSH username for all connections", "  Port:            22")
	sendKeys(m, "j")
	frlShows(t, m, "• User:            x", "> Port:            22", "  ╰ Default SSH port (1–65535)", "  Identity file:   ~/.ssh/id_ed25519")
}

// A disabled field shows its value (or default) without ▾, and its reason
// below it even without focus. A changed field keeps its • when it turns
// disabled.
func TestSettingsDisabledAccent(t *testing.T) {
	for accent, shown := range map[string]string{"": "default", "blue": "blue"} {
		m := frlSettings(func(d *config.Defaults) { d.Colorscheme, d.AccentColor = "nord", accent })
		frlShows(t, m,
			"  Colorscheme:     nord ▾",
			"  Accent color:    "+shown,
			"  ╰ overridden by nord theme",
			"  Field help:      yes ▾")
	}
	m := frlSettings(nil,
		"j", "j", "j", "j", "j", "j", "enter", "j", "enter", // accent: blue
		"k", "enter", "j", "j", "enter") // colorscheme: nord
	frlShows(t, m,
		"> Colorscheme:     nord ▾",
		"  ╰ Application color theme",
		"• Accent color:    blue",
		"  ╰ overridden by nord theme")
}

// A picker value from the config that the options do not list shows as
// it is, trimmed; an empty one shows as default.
func TestSettingsPickerShowsAnUnlistedValue(t *testing.T) {
	m := frlSettings(func(d *config.Defaults) { d.Tmux, d.OpenMode = "", " weird " })
	frlShows(t, m, "  Tmux mode:       default ▾", "  Open mode:       weird ▾")
}

// The group form: picker displays, the sub-modal's default and value, and
// no help line on fields without help text.
func TestGroupFormRowsShowPickersAndSubModal(t *testing.T) {
	m := frlGroupForm(config.Group{Name: "prod"})
	frlShows(t, m, "> Name:          prod", frlSection("SSH", 98))
	frlShows(t, m, "  Open mode:     inherit ▾", "  Tmux:          inherit ▾")
	frlShows(t, m, "  Pane border:   inherit ▾", "  Border format: default")

	m = frlGroupForm(config.Group{Name: "prod", OpenMode: "tmux-window", PaneBorderFmt: "#G"}, "k")
	frlShows(t, m, "  Name:          prod", frlSection("SSH", 98))
	frlShows(t, m, "  Open mode:     tmux-window ▾", "  Tmux:          inherit ▾")
	frlShows(t, m, "  Pane border:   inherit ▾", "> Border format: #G")
}

// Rows fill the width inside the box; the value column keeps at least 10
// columns, a narrow field at most 12. The first row is the first section.
func TestFormContentFillsTheWidth(t *testing.T) {
	r := strings.Repeat
	f := formOf(map[string]string{"name": "box", "port": "", "mode": "auto", "formats": ""},
		fieldDef{Key: "name", Label: "Name", Kind: fieldText},
		fieldDef{Key: "port", Label: "Port", Kind: fieldNumber, Narrow: true},
		fieldDef{Key: "mode", Label: "Mode", Kind: fieldPicker, Options: opts("auto", "never")},
		fieldDef{Key: "formats", Label: "Formats", Kind: fieldSubModal},
	)
	for _, c := range []struct {
		innerW int
		want   []string
	}{
		{40, []string{
			"── S " + r("─", 35),
			"> Name:      box " + r("─", 23),
			"  Port:       " + r("─", 11),
			"  Mode:      auto ▾" + r(" ", 21),
			"  Formats:   default" + r(" ", 20),
		}},
		{20, []string{
			"── S " + r("─", 15),
			"> Name:      box " + r("─", 6),
			"  Port:       " + r("─", 9),
			"  Mode:      auto ▾" + r(" ", 4),
			"  Formats:   default" + r(" ", 3),
		}},
	} {
		got, focusLine := frlLines(f, c.innerW)
		if !slices.Equal(got, c.want) || focusLine != 1 {
			t.Fatalf("innerW %d: focus line %d, rows\n%q\nwant focus 1, rows\n%q", c.innerW, focusLine, got, c.want)
		}
	}
}

// A section without fields draws no header.
func TestFormContentSkipsAnEmptySection(t *testing.T) {
	f := formOf(map[string]string{"a": "1"}, fieldDef{Key: "a", Label: "A", Kind: fieldSubModal})
	f.schema.Sections = append([]sectionDef{{Key: "e", Label: "Empty"}}, f.schema.Sections...)
	f.rebuildItems()
	got, focusLine := frlLines(f, 30)
	want := []string{"── S " + strings.Repeat("─", 25), "> A:         1" + strings.Repeat(" ", 16)}
	if !slices.Equal(got, want) || focusLine != 1 {
		t.Fatalf("focus line %d, rows\n%q\nwant focus 1, rows\n%q", focusLine, got, want)
	}
}

// A field that turns disabled under the focus loses the > mark and shows
// its reason, not its help or error.
func TestFormFocusedFieldTurnedDisabledLosesTheMark(t *testing.T) {
	f := testForm("auto")
	f.showFieldHelp = true
	f.schema.Sections[0].Fields[1].Helper = "port help"
	f.validationErrs["port"] = "bad port"
	formKeys(f, "j")
	f.setValue("mode", "never")
	got, focusLine := frlLines(f, 40)
	want := []string{"  Port:      default", "  ╰ off", "• Mode:      never ▾" + strings.Repeat(" ", 20)}
	if !slices.Equal(got[2:5], want) || focusLine != 2 {
		t.Fatalf("focus line %d, rows\n%q\nwant focus 2, rows 2-4\n%q", focusLine, got, want)
	}
}
