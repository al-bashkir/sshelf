package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// formOf builds a form with one section of the given fields.
func formOf(values map[string]string, fields ...fieldDef) *formModel {
	f := newFormModel(formSchema{Sections: []sectionDef{{Key: "s", Label: "S", Fields: fields}}}, values, 10, false)
	return &f
}

// testForm has each field kind: Name (text, required), Port (number,
// disabled while Mode is never), Mode (picker, rejects force) and Formats
// (sub-modal).
func testForm(mode string) *formModel {
	rejects := func(bad, msg string) func(string) error {
		return func(s string) error {
			if s == bad {
				return errors.New(msg)
			}
			return nil
		}
	}
	return formOf(map[string]string{"name": "box", "port": "", "mode": mode, "formats": ""},
		fieldDef{Key: "name", Label: "Name", Kind: fieldText, Validate: rejects("", "required")},
		fieldDef{Key: "port", Label: "Port", Kind: fieldNumber, DisabledWhen: func(v map[string]string) string {
			if v["mode"] == "never" {
				return "off"
			}
			return ""
		}},
		fieldDef{Key: "mode", Label: "Mode", Kind: fieldPicker, Options: opts("auto", "force", "never"),
			Validate: rejects("force", "not allowed")},
		fieldDef{Key: "formats", Label: "Formats", Kind: fieldSubModal},
	)
}

// formKeys sends keys to f and returns whether the form used the last one.
func formKeys(f *formModel, ks ...string) bool {
	used := false
	for _, k := range ks {
		used, _ = f.handleKey(hk(k))
	}
	return used
}

func wantFocus(t *testing.T, f *formModel, want string) {
	t.Helper()
	if got := f.focusedFieldKey(); got != want {
		t.Fatalf("focus on %q, want %q", got, want)
	}
}

func TestFormNavSkipsDisabledAndWraps(t *testing.T) {
	// Port is disabled while Mode is never: the order is name, mode, formats.
	for _, c := range []struct {
		keys []string
		want []string
	}{
		{[]string{"j", "down", "tab"}, []string{"mode", "formats", "name"}},
		{[]string{"k", "up", "shift+tab"}, []string{"formats", "mode", "name"}},
	} {
		for _, k := range c.keys {
			f := testForm("never")
			for _, want := range c.want {
				if !formKeys(f, k) {
					t.Fatalf("%s not used", k)
				}
				wantFocus(t, f, want)
			}
		}
	}
	f := testForm("auto")
	formKeys(f, "j")
	wantFocus(t, f, "port")
}

func TestFormAllDisabledKeepsFocusAndIgnoresKeys(t *testing.T) {
	off := func(map[string]string) string { return "off" }
	f := formOf(map[string]string{"mode": "", "name": ""},
		fieldDef{Key: "mode", Label: "Mode", Kind: fieldPicker, Options: opts("x", "y"), DisabledWhen: off},
		fieldDef{Key: "name", Label: "Name", Kind: fieldText, DisabledWhen: off},
	)
	for _, k := range []string{"j", "k", "i", "enter", "space", "l"} {
		if !formKeys(f, k) {
			t.Fatalf("%s not used", k)
		}
		wantFocus(t, f, "mode")
		if f.editing || f.picker != nil {
			t.Fatalf("%s acted on a disabled field", k)
		}
	}
}

func TestFormFocusedFieldTurnsDisabled(t *testing.T) {
	for _, k := range []string{"enter", "space"} {
		f := testForm("auto")
		formKeys(f, "j")
		wantFocus(t, f, "port")
		f.setValue("mode", "never") // a change from outside the field, as a wrapper makes
		if !formKeys(f, "i") || f.editing {
			t.Fatal("i edited a disabled field")
		}
		if !formKeys(f, k) || f.editing {
			t.Fatalf("%s edited a disabled field", k)
		}
		wantFocus(t, f, "mode")
	}
}

func TestFormInsertMode(t *testing.T) {
	for _, start := range []string{"i", "enter", "space"} {
		f := testForm("never")
		formKeys(f, start, "x", "j")
		if !f.editing || f.value("name") != "boxxj" {
			t.Fatalf("%s: editing=%v name=%q, want typing into name", start, f.editing, f.value("name"))
		}
		wantFocus(t, f, "name")
		if !formKeys(f, "esc") || f.editing {
			t.Fatalf("%s: Esc did not end insert mode", start)
		}
		wantFocus(t, f, "name")
	}
	// Enter in insert mode ends it, checks the value and moves on.
	f := testForm("never")
	if !formKeys(f, "i", "backspace", "backspace", "backspace", "enter") || f.editing {
		t.Fatal("Enter did not end insert mode")
	}
	wantFocus(t, f, "mode")
	if f.value("name") != "" || f.validationErrs["name"] != "required" {
		t.Fatalf("name=%q err=%q, want empty and required", f.value("name"), f.validationErrs["name"])
	}
}

func TestFormOnlyTextFieldsEdit(t *testing.T) {
	f := testForm("never")
	for range 2 { // mode, then formats
		formKeys(f, "j")
		if !formKeys(f, "i") || f.editing || f.picker != nil {
			t.Fatalf("i on %s: used and nothing else wanted", f.focusedFieldKey())
		}
	}
	formKeys(f, "j") // name
	if !formKeys(f, "l") || !formKeys(f, "right") || f.editing || f.picker != nil {
		t.Fatal("l on a text field: used and nothing else wanted")
	}
	for _, k := range []string{"z", "q", "esc"} {
		if formKeys(f, k) {
			t.Fatalf("%s used, want it left to the wrapper", k)
		}
	}
}

func TestFormLeavesSubModalToWrapper(t *testing.T) {
	for _, k := range []string{"enter", "space", "l", "right"} {
		f := testForm("never")
		formKeys(f, "k")
		wantFocus(t, f, "formats")
		if formKeys(f, k) {
			t.Fatalf("%s used, want it left to the wrapper", k)
		}
	}
}

func TestFormPickerKeys(t *testing.T) {
	for _, open := range []string{"enter", "space", "l", "right"} {
		f := testForm("never")
		formKeys(f, "j")
		if !formKeys(f, open) || f.picker == nil {
			t.Fatalf("%s did not open the picker", open)
		}
		if got := f.picker.selected(); got != "never" {
			t.Fatalf("picker opened on %q, want the current value never", got)
		}
	}

	f := testForm("never")
	formKeys(f, "j", "enter")
	// The popup takes every key; j/down and k/up move and wrap.
	for _, s := range []struct{ key, want string }{
		{"j", "auto"}, {"down", "force"}, {"k", "auto"}, {"up", "never"}, {"x", "never"}, {"i", "never"},
	} {
		if !formKeys(f, s.key) || f.picker == nil {
			t.Fatalf("%s: picker closed or key not used", s.key)
		}
		if got := f.picker.selected(); got != s.want {
			t.Fatalf("%s: cursor on %q, want %q", s.key, got, s.want)
		}
	}
	wantFocus(t, f, "mode")

	// Esc closes and keeps the value.
	if !formKeys(f, "j", "esc") || f.picker != nil || f.value("mode") != "never" {
		t.Fatalf("Esc: picker=%v mode=%q, want closed and never", f.picker != nil, f.value("mode"))
	}
	// Enter stores the choice and checks it.
	formKeys(f, "enter", "j", "j", "enter")
	if f.picker != nil || f.value("mode") != "force" || f.validationErrs["mode"] != "not allowed" {
		t.Fatalf("mode=%q err=%q, want force and not allowed", f.value("mode"), f.validationErrs["mode"])
	}
	formKeys(f, "enter", "k", "enter")
	if f.value("mode") != "auto" || f.validationErrs["mode"] != "" {
		t.Fatalf("mode=%q err=%q, want auto and no error", f.value("mode"), f.validationErrs["mode"])
	}
	formKeys(f, "k") // Port is enabled again.
	wantFocus(t, f, "port")
}

func TestFormPickerWithoutOptionsStaysClosed(t *testing.T) {
	f := formOf(map[string]string{"p": ""}, fieldDef{Key: "p", Label: "P", Kind: fieldPicker})
	if !formKeys(f, "enter") || f.picker != nil {
		t.Fatal("a picker without options opened")
	}
}

func TestFormFooter(t *testing.T) {
	footer := func(f *formModel) string { return ansi.Strip(f.renderFooter()) }
	f := testForm("never")
	for _, s := range []struct{ key, want string }{
		{"", "1/4 S  Ctrl+S save  j/k nav  i edit  Esc back"},
		{"i", "1/4 S  INSERT  Ctrl+S save  Esc done"},
		{"esc", "1/4 S  Ctrl+S save  j/k nav  i edit  Esc back"},
		{"j", "3/4 S  Ctrl+S save  j/k nav  ⏎ choose  Esc back"},
		{"enter", "3/4 S  ⏎ select  Esc cancel"},
		{"esc", "3/4 S  Ctrl+S save  j/k nav  ⏎ choose  Esc back"},
		{"j", "4/4 S  Ctrl+S save  j/k nav  ⏎ select  Esc back"},
	} {
		if s.key != "" {
			formKeys(f, s.key)
		}
		if got := footer(f); got != s.want {
			t.Fatalf("after %q footer %q, want %q", s.key, got, s.want)
		}
	}
	if got := footer(formOf(map[string]string{})); got != "  Ctrl+S save  j/k nav  Esc back" {
		t.Fatalf("empty form footer %q", got)
	}
}

// overlayRow is the plain text of bg with the popup line drawn from col.
func overlayRow(bg, line string, col int) string {
	plain := ansi.Strip(line)
	padded := bg + strings.Repeat(" ", max(0, col-len(bg)))
	right := ""
	if end := col + lipgloss.Width(plain); end < len(padded) {
		right = padded[end:]
	}
	return padded[:col] + plain + right
}

func TestPickerOverlayDropsBelowOrFlipsAbove(t *testing.T) {
	rows := func(n int, s string) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = strings.Repeat(string(rune('a'+i)), len(s)) + s
		}
		return out
	}
	f := testForm("never")
	if got := f.overlayPickerOnVisible(rows(3, "x"), 0, 40); strings.Join(got, "\n") != strings.Join(rows(3, "x"), "\n") {
		t.Fatal("no picker open, but the rows changed")
	}
	formKeys(f, "j", "enter")
	popup := strings.Split(f.picker.View(28), "\n") // 5 lines, 22 wide
	const col = 13                                  // indent 2 + label 10 + 1
	for _, c := range []struct {
		name            string
		bg              []string
		focusRow, first int
	}{
		{"room below", rows(10, strings.Repeat("-", 40)), 1, 2},
		{"flip above", rows(10, strings.Repeat("-", 40)), 8, 3},
		{"little room either way", rows(6, strings.Repeat("-", 40)), 2, 3},
		{"flip above, top cut", rows(7, strings.Repeat("-", 40)), 4, -1},
		{"short rows", rows(10, ""), 1, 2},
	} {
		got := f.overlayPickerOnVisible(c.bg, c.focusRow, 40)
		if len(got) != len(c.bg) {
			t.Fatalf("%s: %d rows, want %d", c.name, len(got), len(c.bg))
		}
		for r := range got {
			want := c.bg[r]
			if i := r - c.first; i >= 0 && i < len(popup) {
				want = overlayRow(c.bg[r], popup[i], col)
			}
			if plain := ansi.Strip(got[r]); plain != want {
				t.Fatalf("%s: row %d\n got %q\nwant %q", c.name, r, plain, want)
			}
		}
	}
	// A narrow form gives the popup at least 20 columns, or all it has.
	for innerW, wantW := range map[int]int{25: 20, 15: 15} {
		got := f.overlayPickerOnVisible(rows(8, ""), 0, innerW)
		if w := lipgloss.Width(ansi.Strip(got[1])); w != col+wantW {
			t.Fatalf("innerW %d: popup %d wide, want %d", innerW, w-col, wantW)
		}
	}
}

func TestOptionPickerView(t *testing.T) {
	view := func(p *optionPickerPopup, maxW int) string { return ansi.Strip(p.View(maxW)) }
	modes := opts("auto", "force", "never")
	for _, c := range []struct {
		name string
		p    *optionPickerPopup
		maxW int
		want string
	}{
		{"at least 20 inside, cursor on the value", newOptionPickerPopup("mode", "Mode", " force ", modes), 0, `┌ Mode ──────────────┐
│  auto              │
│▸ force             │
│  never             │
└────────────────────┘`},
		{"unknown value puts the cursor first", newOptionPickerPopup("mode", "Mode", "nope", modes), 0, `┌ Mode ──────────────┐
│▸ auto              │
│  force             │
│  never             │
└────────────────────┘`},
		{"widest option sets the width", newOptionPickerPopup("k", "K", "", opts("a-very-long-option-name", "b")), 0, `┌ K ────────────────────────┐
│▸ a-very-long-option-name  │
│  b                        │
└───────────────────────────┘`},
		{"a long label sets the width", newOptionPickerPopup("k", "A long label for this one", "", opts("a")), 0, `┌ A long label for this one ──┐
│▸ a                          │
└─────────────────────────────┘`},
		{"maxW cuts the box", newOptionPickerPopup("k", "K", "b", opts("a-very-long-option-name", "b", "c")), 12, `┌ K ───────┐
│  a-very-l│
│▸ b       │
│  c       │
└──────────┘`},
		{"maxW cuts the cursor row", newOptionPickerPopup("k", "K", "", opts("a-very-long-option-name")), 12, `┌ K ───────┐
│▸ a-very-l│
└──────────┘`},
	} {
		if got := view(c.p, c.maxW); got != c.want {
			t.Fatalf("%s:\n%s\nwant\n%s", c.name, got, c.want)
		}
	}
}
