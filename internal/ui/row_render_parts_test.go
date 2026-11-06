package ui

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"github.com/al-bashkir/sshelf/internal/config"
)

// rowpCase is one rendered row and the text (or styled string) it must be.
type rowpCase struct {
	name, got, want string
}

// rowpCheck fails each case whose rendered string is not the wanted one.
// plain compares the text only, without style codes.
func rowpCheck(t *testing.T, plain bool, cases []rowpCase) {
	t.Helper()
	for _, c := range cases {
		got := c.got
		if plain {
			got = ansi.Strip(got)
		}
		if got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// rowpPad fills s with spaces to w cells, as an active row is filled.
func rowpPad(s string, w int) string {
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func TestRowpTruncateTail(t *testing.T) {
	rowpCheck(t, true, []rowpCase{
		{"one cell spare", truncateTail("alpha", 6), "alpha"},
		{"exact width", truncateTail("alpha", 5), "alpha"},
		{"one cell short", truncateTail("alpha", 4), "alp…"},
		{"two cells", truncateTail("alpha", 2), "a…"},
		{"one cell", truncateTail("alpha", 1), "…"},
		{"zero", truncateTail("alpha", 0), ""},
		{"negative", truncateTail("alpha", -1), ""},
		{"empty", truncateTail("", 3), ""},
		{"multi-byte fits", truncateTail("héllo↵", 6), "héllo↵"},
		{"multi-byte cut on a rune", truncateTail("héllo↵", 5), "héll…"},
		{"accent mark fits", truncateTail("cafe\u0301", 4), "cafe\u0301"},
		{"one wide rune in one cell", truncateTail("日", 1), "…"},
	})
}

func TestRowpTruncateFade(t *testing.T) {
	rowpCheck(t, true, []rowpCase{
		{"one cell spare", truncateFade("alpha", 6), "alpha"},
		{"exact width", truncateFade("alpha", 5), "alpha"},
		{"one cell short", truncateFade("alpha", 4), "alp…"},
		{"three cells", truncateFade("alpha", 3), "al…"},
		{"two cells", truncateFade("alpha", 2), "… "},
		{"one cell", truncateFade("alpha", 1), "…"},
		{"zero", truncateFade("alpha", 0), ""},
		{"negative", truncateFade("alpha", -2), ""},
		{"empty", truncateFade("", 3), ""},
		{"multi-byte cut on a rune", truncateFade("héllo↵", 5), "héll…"},
		{"accent mark fits", truncateFade("cafe\u0301", 4), "cafe\u0301"},
	})
}

// Wide letters (CJK, emoji) take two cells, so a cut counts cells, not
// runes, and keeps an emoji made of several runes whole.
func TestRowpCutsWideLettersByWidth(t *testing.T) {
	rowpCheck(t, true, []rowpCase{
		{"tail fits by runes, not cells", truncateTail("日本語", 4), "日…"},
		{"tail", truncateTail("日本語日本", 5), "日本…"},
		{"tail keeps an emoji whole", truncateTail("ab👨‍👩‍👧cd", 4), "ab…"},
		{"fade", truncateFade("日本語のホスト", 7), "日本語…"},
		{"fade fits by runes, not cells", truncateFade("日本語", 5), "日本…"},
		{"fade on a wide letter", truncateFade("a日本", 4), "a日…"},
		{"fade drops a wide letter that does not fit", truncateFade("ab日本", 4), "ab…"},
		{"fade keeps an emoji whole", truncateFade("ab👨‍👩‍👧cd", 5), "ab👨‍👩‍👧…"},
	})
	g := groupsScreen(func(o *Options) {
		o.Inventory.Groups = append(o.Inventory.Groups, config.Group{Name: "日本語のグループの名前"})
	})
	g.Update(tea.WindowSizeMsg{Width: 24, Height: 20})
	rowpCheck(t, false, []rowpCase{{"screen row", rowpRow(t, g, "日本"), "  日本語のグルー…  0"}})
	for _, l := range rowpLines(g) {
		if w := lipgloss.Width(l); w > 24 {
			t.Fatalf("line is %d cells, want <= 24: %q", w, l)
		}
	}
}

func TestRowpComposeRow(t *testing.T) {
	row := func(width int, active bool, prefix, suffix string, suffixW int) string {
		return composeRow(width, active, prefix, "alpha", suffix, suffixW, nil, false)
	}
	rowpCheck(t, true, []rowpCase{
		{"inactive row is not filled", row(12, false, "  ", "", 0), "  alpha"},
		{"active row is filled to the width", row(12, true, "▸ ", "", 0), rowpPad("▸ alpha", 12)},
		{"active row one cell spare", row(8, true, "▸ ", "", 0), "▸ alpha "},
		{"exact width", row(7, false, "  ", "", 0), "  alpha"},
		{"exact width active", row(7, true, "▸ ", "", 0), "▸ alpha"},
		{"one cell short fades", row(6, false, "  ", "", 0), "  alp…"},
		{"one cell short active cuts", row(6, true, "▸ ", "", 0), "▸ alp…"},
		{"suffix follows the text", row(12, false, "  ", "[x]", 3), "  alpha[x]"},
		{"suffix exact width", row(10, false, "  ", "[x]", 3), "  alpha[x]"},
		{"text cut for the suffix", row(9, false, "  ", "[x]", 3), "  alp…[x]"},
		{"suffix fills what the prefix leaves", row(5, false, "  ", "[x]", 3), "  [x]"},
		{"suffix too wide is dropped", row(4, false, "  ", "[x]", 3), "  … "},
		{"suffix too wide is dropped, active", row(4, true, "▸ ", "[x]", 3), "▸ a…"},
		{"suffix uses its given width", row(10, false, "  ", "[x]", 5), "  al…[x]"},
		{"width 3", row(3, false, "  ", "", 0), "  …"},
		{"width 3 active", row(3, true, "▸ ", "", 0), "▸ …"},
		{"width 2", row(2, false, "  ", "", 0), "  "},
		{"width 1 drops the text", row(1, false, "  ", "[x]", 3), "  "},
		{"width 1 active", row(1, true, "▸ ", "", 0), "▸ "},
		{"width 0 does not cut or fill", row(0, true, "▸ ", "[x]", 3), "▸ alpha[x]"},
		{"negative width does not cut", row(-1, false, "  ", "[x]", 3), "  alpha[x]"},
		{"empty active row fills width 1", composeRow(1, true, "", "", "", 0, nil, false), " "},
		{"dim text keeps its text", composeRow(12, false, "  ", "alpha", "", 0, nil, true), "  alpha"},
		{"matches keep the text", composeRow(12, false, "  ", "alpha", "", 0, []int{0, 4, 9, -1}, false), "  alpha"},
		{"matches keep the cut text", composeRow(6, false, "  ", "alpha", "", 0, []int{0, 3, 4}, false), "  alp…"},
	})
}

func TestRowpHostLikeRow(t *testing.T) {
	host := func(width int, active, selected, hasCfg, hidden bool) string {
		return renderHostLikeRow(width, active, selected, "alpha", hasCfg, hidden, []int{0})
	}
	rowpCheck(t, true, []rowpCase{
		{"inactive", host(20, false, false, false, false), "  ◻ alpha"},
		{"active", host(20, true, false, false, false), rowpPad("▸ ◻ alpha", 20)},
		{"selected", host(20, false, true, false, false), "  ◼ alpha"},
		{"selected active", host(20, true, true, false, false), rowpPad("▸ ◼ alpha", 20)},
		{"config badge", host(20, false, false, true, false), "  ◻ alpha  ⚙ "},
		{"config badge active", host(20, true, true, true, false), rowpPad("▸ ◼ alpha  ⚙ ", 20)},
		{"hidden", host(20, false, false, false, true), "  ◻ ⊘ alpha"},
		{"hidden active", host(20, true, false, false, true), rowpPad("▸ ◻ ⊘ alpha", 20)},
		{"exact width", host(9, false, false, false, false), "  ◻ alpha"},
		{"one cell short", host(8, false, false, false, false), "  ◻ alp…"},
		{"badge exact width", host(13, false, false, true, false), "  ◻ alpha  ⚙ "},
		{"badge one cell short", host(12, false, false, true, false), "  ◻ alp…  ⚙ "},
		{"badge one cell short active", host(12, true, false, true, false), "▸ ◻ alp…  ⚙ "},
		{"badge too wide is dropped", host(7, false, false, true, false), "  ◻ al…"},
		{"hidden one cell short", host(10, false, false, false, true), "  ◻ ⊘ alp…"},
	})
}

func TestRowpGroupAndSimpleRows(t *testing.T) {
	rowpCheck(t, true, []rowpCase{
		{"group", renderGroupRow(20, false, "db", 0, 2, nil), "  db  2 "},
		{"group active", renderGroupRow(20, true, "db", 0, 2, []int{0}), rowpPad("▸ db  2 ", 20)},
		{"group big count", renderGroupRow(20, false, "web", 0, 123, nil), "  web  123 "},
		{"group exact width", renderGroupRow(8, false, "db", 0, 2, nil), "  db  2 "},
		{"group long name exact width", renderGroupRow(11, false, "alpha", 0, 2, nil), "  alpha  2 "},
		{"group one cell short", renderGroupRow(10, false, "alpha", 0, 2, nil), "  alp…  2 "},
		{"group badge too wide", renderGroupRow(5, false, "alpha", 0, 2, nil), "  al…"},
		{"simple", renderSimpleRow(20, false, "alpha", nil), "  alpha"},
		{"simple active", renderSimpleRow(20, true, "alpha", []int{1}), rowpPad("▸ alpha", 20)},
		{"simple exact width", renderSimpleRow(7, false, "alpha", nil), "  alpha"},
		{"simple one cell short", renderSimpleRow(6, true, "alpha", nil), "▸ alp…"},
	})
}

// rowpLines is the plain view of m, one string per line.
func rowpLines(m tea.Model) []string { return strings.Split(plainView(m), "\n") }

// rowpRow is the first line of the view that has s, trimmed of the frame.
func rowpRow(t *testing.T, m tea.Model, s string) string {
	t.Helper()
	for _, l := range rowpLines(m) {
		if strings.Contains(l, s) {
			return strings.TrimRight(strings.Trim(l, "│"), " ")
		}
	}
	t.Fatalf("no line has %q:\n%s", s, plainView(m))
	return ""
}

func TestRowpScreens(t *testing.T) {
	m := hostsScreen(func(o *Options) {
		o.Inventory = config.Inventory{Hosts: []config.Host{{Host: "beta", User: "u"}, {Host: "gamma", Hidden: true}}}
	})
	rowpCheck(t, false, []rowpCase{
		{"cursor host", rowpRow(t, m, "alpha"), "▸ ◻ alpha"},
		{"host with config", rowpRow(t, m, "beta"), "  ◻ beta  ⚙"},
	})
	screenShows(t, m, nil, "gamma")
	on(m, "H")
	rowpCheck(t, false, []rowpCase{{"hidden host", rowpRow(t, m, "gamma"), "  ◻ ⊘ gamma  ⚙"}})
	on(m, "space")
	rowpCheck(t, false, []rowpCase{{"selected host", rowpRow(t, m, "alpha"), "▸ ◼ alpha"}})
	on(m, "/", "b")
	rowpCheck(t, false, []rowpCase{{"filtered host", rowpRow(t, m, "beta"), "▸ ◻ beta  ⚙"}})

	g := groupsScreen(nil)
	rowpCheck(t, false, []rowpCase{
		{"cursor group", rowpRow(t, g, "db"), "▸ db  2"},
		{"empty group", rowpRow(t, g, "empty"), "  empty  0"},
		{"group", rowpRow(t, g, "web"), "  web  1"},
	})
}

// A narrow window cuts long names: the cursor row with a hard "…", the
// others with a fade. Every line keeps to the window width.
func TestRowpScreensCutLongNames(t *testing.T) {
	m := hostsScreen(func(o *Options) {
		o.Hosts = []string{"alpha-very-long-host-name", "beta-very-long-host-name"}
		o.Inventory = config.Inventory{Hosts: []config.Host{{Host: "beta-very-long-host-name", User: "u"}}}
	})
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 20})
	rowpCheck(t, false, []rowpCase{
		{"cursor host", rowpRow(t, m, "alpha"), "▸ ◻ alpha-very-long-h…"},
		{"host with config", rowpRow(t, m, "beta"), "  ◻ beta-very-lon…  ⚙"},
	})
	for _, l := range rowpLines(m) {
		if w := lipgloss.Width(l); w > 24 {
			t.Fatalf("line is %d cells, want <= 24: %q", w, l)
		}
	}
}

// rowpInColour reports whether this is the colour run of test t. Styles
// render as plain text unless the output is a terminal, so outside that run
// it runs t again in a child process with CLICOLOR_FORCE=1 and fails t if
// the child fails.
func rowpInColour(t *testing.T) bool {
	t.Helper()
	if os.Getenv("ROWP_COLOUR_CHILD") == "1" {
		if checkedStyle.Render("x") == "x" {
			t.Fatal("CLICOLOR_FORCE=1 gave no colour")
		}
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), "CLICOLOR_FORCE=1", "NO_COLOR=", "ROWP_COLOUR_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("colour run failed: %v\n%s", err, out)
	}
	return false
}

// rowpMarks is the fuzzy match of query on s: byte offsets of the matched
// letters.
func rowpMarks(query, s string) []int {
	return fuzzy.Find(query, []string{s})[0].MatchedIndexes
}

// Fuzzy matches give byte offsets, so in a name with multi-byte letters the
// marks must land on the letters the query matched.
func TestRowpColourMarksFuzzyLettersOfMultiByteNames(t *testing.T) {
	if !rowpInColour(t) {
		return
	}
	hl, ul := checkedStyle.Render, lipgloss.NewStyle().Underline(true).Render
	rowpCheck(t, false, []rowpCase{
		{"accent before the match", renderGroupRow(0, false, "café-web", 0, 1, rowpMarks("web", "café-web")),
			"  café-" + hl("web") + " " + badgeCountStyle.Render("1")},
		{"cyrillic", renderSimpleRow(0, false, "прод", rowpMarks("од", "прод")), "  пр" + hl("од")},
		{"around an arrow", renderSimpleRow(0, false, "a↵b", rowpMarks("ab", "a↵b")), "  " + hl("a") + "↵" + hl("b")},
		{"offset inside a rune", highlightMatches("héllo", []int{2}, false), "héllo"},
	})
	g := groupsScreen(func(o *Options) { o.Inventory.Groups = append(o.Inventory.Groups, config.Group{Name: "прод"}) })
	on(g, "/", "о", "д")
	if v := g.View(); !strings.Contains(v, "▸ пр"+ul("о")+rowpOpen(rowActiveStyle)+ul("д")) {
		t.Fatalf("groups screen does not mark %q in the row:\n%q", "од", v)
	}
}

// A match on a letter the fade cut or dimmed must not mark inside the
// fade's escape code: that printed the code as text ("[90m") and made the
// row too wide.
func TestRowpColourMatchInTheFade(t *testing.T) {
	if !rowpInColour(t) {
		return
	}
	hl := checkedStyle.Render
	rowpCheck(t, false, []rowpCase{
		{"on the dim letter", composeRow(6, false, "  ", "alpha", "", 0, []int{2}, false), "  al" + dim.Render("p…")},
		{"past the cut", composeRow(6, false, "  ", "alpha", "", 0, []int{1, 3, 4}, false), "  a" + hl("l") + dim.Render("p…")},
	})
	m := hostsScreen(func(o *Options) { o.Hosts = []string{"alpha-very-long-host-name", "beta-very-long-host-name"} })
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 20})
	on(m, "/", "name")
	rowpCheck(t, false, []rowpCase{{"faded screen row", rowpRow(t, m, "beta"), "  ◻ beta-very-long-ho…"}})
	for _, l := range rowpLines(m) {
		if w := lipgloss.Width(l); w > 24 {
			t.Fatalf("line is %d cells, want <= 24: %q", w, l)
		}
	}
}

// TestRowpColour checks the style codes of the rows.
func TestRowpColour(t *testing.T) {
	if !rowpInColour(t) {
		return
	}
	hl, ul := checkedStyle.Render, lipgloss.NewStyle().Underline(true).Render
	act := func(s string, w int) string { return rowActiveStyle.Render(rowpPad(s, w)) }
	bar := rowpOpen(rowActiveStyle) // the active row sets the bar again after each reset
	box, tick := uncheckedStyle.Render("◻"), checkedStyle.Render("◼")
	cfg, two := " "+badgeCfgStyle.Render("⚙"), " "+badgeCountStyle.Render("2")
	rowpCheck(t, false, []rowpCase{
		{"no matches", highlightMatches("alpha", nil, false), "alpha"},
		{"empty matches", highlightMatches("alpha", []int{}, true), "alpha"},
		{"first letter", highlightMatches("alpha", []int{0}, false), hl("a") + "lpha"},
		{"last letter", highlightMatches("alpha", []int{4}, false), "alph" + hl("a")},
		{"two runs", highlightMatches("alpha", []int{0, 1, 4}, false), hl("al") + "ph" + hl("a")},
		{"inner run", highlightMatches("alpha", []int{1, 2, 3}, false), "a" + hl("lph") + "a"},
		{"every letter", highlightMatches("alpha", []int{0, 1, 2, 3, 4}, false), hl("alpha")},
		{"out of range", highlightMatches("alpha", []int{-1, 5, 99}, false), "alpha"},
		{"twice", highlightMatches("alpha", []int{2, 2}, false), "al" + hl("p") + "ha"},
		{"unsorted", highlightMatches("alpha", []int{3, 1}, false), "a" + hl("l") + "p" + hl("h") + "a"},
		{"empty text", highlightMatches("", []int{0}, false), ""},
		{"active underlines", highlightMatches("alpha", []int{0, 4}, true), ul("a") + "lph" + ul("a")},

		{"fade is dim", truncateFade("alpha", 4), "al" + dim.Render("p…")},
		{"short fade is dim", truncateFade("alpha", 2), dim.Render("…") + " "},

		{"row match", composeRow(20, false, "  ", "alpha", "", 0, []int{0}, false), "  " + hl("a") + "lpha"},
		{"row out of range matches", composeRow(20, false, "  ", "alpha", "", 0, []int{-1, 5}, false), "  alpha"},
		{"row match before the fade", composeRow(6, false, "  ", "alpha", "", 0, []int{0}, false),
			"  " + hl("a") + "l" + dim.Render("p…")},
		{"active row match", composeRow(20, true, "▸ ", "alpha", "", 0, []int{0}, false), act("▸ "+ul("a")+bar+"lpha", 20)},
		{"active row match on the cut", composeRow(6, true, "▸ ", "alpha", "", 0, []int{2, 3}, false), act("▸ al"+ul("p")+bar+ul("…")+bar, 6)},
		{"dim row", composeRow(20, false, "  ", "alpha", "", 0, nil, true), "  " + dim.Render("alpha")},
		{"dim active row", composeRow(20, true, "▸ ", "alpha", "", 0, nil, true), act("▸ "+dim.Render("alpha")+bar, 20)},

		{"host", renderHostLikeRow(20, false, false, "alpha", false, false, []int{0}), "  " + box + " " + hl("a") + "lpha"},
		{"host selected", renderHostLikeRow(20, false, true, "alpha", false, false, nil), "  " + tick + " alpha"},
		{"host config", renderHostLikeRow(20, false, false, "alpha", true, false, nil), "  " + box + " alpha" + cfg},
		{"host hidden", renderHostLikeRow(20, false, false, "alpha", false, true, []int{0}), "  " + box + " " + dim.Render("⊘ alpha")},
		{"host active", renderHostLikeRow(20, true, true, "alpha", true, false, []int{1}), act("▸ ◼ a"+ul("l")+bar+"pha  ⚙ ", 20)},
		{"host hidden active", renderHostLikeRow(20, true, false, "alpha", false, true, []int{0}), act("▸ ◻ ⊘ alpha", 20)},

		{"group", renderGroupRow(20, false, "db", 0, 2, []int{0}), "  " + hl("d") + "b" + two},
		{"group active", renderGroupRow(20, true, "db", 0, 2, []int{1}), act("▸ d"+ul("b")+bar+"  2 ", 20)},
		{"simple", renderSimpleRow(20, false, "alpha", []int{2}), "  al" + hl("p") + "ha"},
		{"simple active", renderSimpleRow(20, true, "alpha", nil), act("▸ alpha", 20)},
	})
}

// rowpCell is one cell a terminal draws: its text and the SGR codes set
// since the last reset.
type rowpCell struct{ text, sgr string }

// rowpCells decodes s into the cells a terminal draws.
func rowpCells(s string) []rowpCell {
	var cells []rowpCell
	var sgr string
	var state byte
	for s != "" {
		seq, w, n, next := ansi.DecodeSequence(s, state, nil)
		switch {
		case w > 0:
			cells = append(cells, rowpCell{seq, sgr})
		case seq == "\x1b[0m" || seq == "\x1b[m":
			sgr = ""
		case strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m"):
			sgr += seq
		}
		s, state = s[n:], next
	}
	return cells
}

// rowpOpen is the SGR code that st starts its text with.
func rowpOpen(st lipgloss.Style) string {
	open, _, _ := strings.Cut(st.Render("x"), "x")
	return open
}

// rowpBarCells fails t when a cell is off the active row's bar, and gives
// the text of the underlined cells.
func rowpBarCells(t *testing.T, name string, cells []rowpCell) string {
	t.Helper()
	bar, ul := rowpOpen(rowActiveStyle), rowpOpen(lipgloss.NewStyle().Underline(true))
	var marked string
	for i, c := range cells {
		if !strings.Contains(c.sgr, bar) {
			t.Errorf("%s: cell %d %q is off the bar %q: it has %q", name, i, c.text, bar, c.sgr)
			return marked
		}
		if strings.Contains(c.sgr, ul) {
			marked += c.text
		}
	}
	return marked
}

// The cursor row is one solid bar: every cell to the full width, the
// marked letters too, has the bar's colours. The reset that ended a marked
// letter used to end the bar there too.
func TestRowpColourActiveRowKeepsTheBar(t *testing.T) {
	if !rowpInColour(t) {
		return
	}
	for _, c := range []struct {
		name, row, marked string
		width             int
	}{
		{"no marks", renderSimpleRow(20, true, "alpha", nil), "", 20},
		{"two marks", renderSimpleRow(20, true, "alpha", []int{0, 4}), "aa", 20},
		{"mark before the badge", renderGroupRow(20, true, "db", 0, 2, []int{1}), "b", 20},
		{"host with a badge", renderHostLikeRow(20, true, true, "alpha", true, false, []int{1}), "l", 20},
		{"hidden host drops marks", renderHostLikeRow(20, true, false, "alpha", false, true, []int{0}), "", 20},
		{"mark on the cut", composeRow(6, true, "▸ ", "alpha", "", 0, []int{2, 3}, false), "p…", 6},
		{"cyrillic", renderSimpleRow(20, true, "прод", rowpMarks("од", "прод")), "од", 20},
		{"wide letters", renderSimpleRow(20, true, "日本語", rowpMarks("本", "日本語")), "本", 20},
	} {
		if w := lipgloss.Width(c.row); w != c.width {
			t.Errorf("%s: row is %d cells, want %d", c.name, w, c.width)
		}
		if got := rowpBarCells(t, c.name, rowpCells(c.row)); got != c.marked {
			t.Errorf("%s: marked %q, want %q\nrow %q", c.name, got, c.marked, c.row)
		}
	}

	g := groupsScreen(nil)
	on(g, "/", "d", "b")
	for _, l := range strings.Split(g.View(), "\n") {
		cells := rowpCells(l)
		at := slices.IndexFunc(cells, func(c rowpCell) bool { return c.text == "▸" })
		if at < 0 {
			continue
		}
		row := cells[at:]
		if end := slices.IndexFunc(row, func(c rowpCell) bool { return c.text == "│" }); end >= 0 {
			row = row[:end]
		}
		if got := rowpBarCells(t, "groups screen", row); got != "db" || len(row) < 20 {
			t.Errorf("groups screen: marked %q in %d cells, want %q in 20 or more\nline %q", got, len(row), "db", l)
		}
		return
	}
	t.Fatalf("groups screen has no cursor row:\n%s", plainView(g))
}
