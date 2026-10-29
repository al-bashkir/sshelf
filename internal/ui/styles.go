package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	defaultAccent = lipgloss.AdaptiveColor{Light: "25", Dark: "39"} // blue/cyan
	cAccent       = defaultAccent
	cMuted        = lipgloss.AdaptiveColor{Light: "242", Dark: "242"} // gray
	cOK           = lipgloss.AdaptiveColor{Light: "28", Dark: "35"}
	cWarn         = lipgloss.AdaptiveColor{Light: "166", Dark: "214"}
	cErr          = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}

	cSearchDim   = lipgloss.AdaptiveColor{Light: "247", Dark: "246"}
	cFrameBorder = lipgloss.AdaptiveColor{Light: "250", Dark: "238"}

	// Active list row: solid background bar with contrasting text.
	cRowActiveBG = lipgloss.AdaptiveColor{Light: "253", Dark: "238"}
	cRowActiveFG = lipgloss.AdaptiveColor{Light: "0", Dark: "255"}

	// Extended color roles.
	cFocusedBorder = lipgloss.AdaptiveColor{Light: "25", Dark: "39"}   // = accent by default
	cDisabled      = lipgloss.AdaptiveColor{Light: "248", Dark: "240"} // dimmer than muted
	cHint          = lipgloss.AdaptiveColor{Light: "245", Dark: "245"} // between muted and disabled
	cSecondary     = lipgloss.AdaptiveColor{Light: "67", Dark: "67"}   // secondary accent

	// cBackground is the hex background color for the active colorscheme.
	// Empty string means no override — terminal default is used.
	cBackground string

	// bgANSICode is the precomputed ANSI escape sequence for cBackground,
	// e.g. "\x1b[48;2;40;42;54m". Empty when no theme background is active.
	bgANSICode string
)

// tabBoxBorderStyle styles box-drawing characters (│ ─ ┌ ┐ └ ┘ ├ ┤).
// Rebuilt by applyScheme and SetAccentColor.
var tabBoxBorderStyle = lipgloss.NewStyle().Foreground(cFrameBorder)

// focusedBoxBorderStyle styles box-drawing characters for focused modals/popups.
var focusedBoxBorderStyle = lipgloss.NewStyle().Foreground(cFocusedBorder)

// tabBoxPadStyle fills interior line padding with the theme background.
// Rebuilt by applyScheme and SetAccentColor.
var tabBoxPadStyle = lipgloss.NewStyle()

// colorScheme holds the full set of color variables for a named theme.
// Background is an optional hex color applied to modal/frame backgrounds;
// empty string means use terminal default (no override).
type colorScheme struct {
	Accent      lipgloss.AdaptiveColor
	Muted       lipgloss.AdaptiveColor
	OK          lipgloss.AdaptiveColor
	Warn        lipgloss.AdaptiveColor
	Err         lipgloss.AdaptiveColor
	FrameBorder lipgloss.AdaptiveColor
	SearchDim   lipgloss.AdaptiveColor
	RowActiveBG lipgloss.AdaptiveColor
	RowActiveFG lipgloss.AdaptiveColor
	Background  string // hex color for frame/modal background; "" = terminal default

	// Extended roles.
	FocusedBorder lipgloss.AdaptiveColor // border of focused panel/modal
	Disabled      lipgloss.AdaptiveColor // disabled/unavailable elements
	Hint          lipgloss.AdaptiveColor // helper/placeholder text
	Secondary     lipgloss.AdaptiveColor // secondary accent (section headers, breadcrumbs)
	Surface       lipgloss.AdaptiveColor // elevated background (modals, cards)
}

// themeColor creates an AdaptiveColor from a single truecolor hex value.
// Named themes are dark-themed so the same hex is used for both light and dark modes.
func themeColor(hex string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: hex, Dark: hex}
}

// builtinColorSchemes maps scheme names to their full color definitions.
// Named theme colors are sourced from official palette specifications.
// The "default" entry uses adaptive ANSI codes for proper light/dark terminal support.
var builtinColorSchemes = map[string]colorScheme{
	"default": {
		// Adaptive ANSI 256 codes — no background override.
		Accent:        lipgloss.AdaptiveColor{Light: "25", Dark: "39"},
		Muted:         lipgloss.AdaptiveColor{Light: "242", Dark: "242"},
		OK:            lipgloss.AdaptiveColor{Light: "28", Dark: "35"},
		Warn:          lipgloss.AdaptiveColor{Light: "166", Dark: "214"},
		Err:           lipgloss.AdaptiveColor{Light: "160", Dark: "203"},
		FrameBorder:   lipgloss.AdaptiveColor{Light: "250", Dark: "238"},
		SearchDim:     lipgloss.AdaptiveColor{Light: "247", Dark: "246"},
		RowActiveBG:   lipgloss.AdaptiveColor{Light: "253", Dark: "238"},
		RowActiveFG:   lipgloss.AdaptiveColor{Light: "0", Dark: "255"},
		Background:    "",
		FocusedBorder: lipgloss.AdaptiveColor{Light: "25", Dark: "39"},   // = accent
		Disabled:      lipgloss.AdaptiveColor{Light: "248", Dark: "240"}, // lighter gray
		Hint:          lipgloss.AdaptiveColor{Light: "245", Dark: "245"}, // mid gray
		Secondary:     lipgloss.AdaptiveColor{Light: "67", Dark: "67"},   // steel blue
		Surface:       lipgloss.AdaptiveColor{Light: "254", Dark: "236"}, // raised bg
	},
	// Dracula — https://draculatheme.com/contribute
	"dracula": {
		Accent:        themeColor("#bd93f9"), // purple
		Muted:         themeColor("#6272a4"), // comment
		OK:            themeColor("#50fa7b"), // green
		Warn:          themeColor("#ffb86c"), // orange
		Err:           themeColor("#ff5555"), // red
		FrameBorder:   themeColor("#44475a"), // current line
		SearchDim:     themeColor("#6272a4"), // comment
		RowActiveBG:   themeColor("#44475a"), // selection
		RowActiveFG:   themeColor("#f8f8f2"), // foreground
		Background:    "#282a36",             // background
		FocusedBorder: themeColor("#bd93f9"), // purple (= accent)
		Disabled:      themeColor("#383a59"), // darker than current line
		Hint:          themeColor("#6272a4"), // comment
		Secondary:     themeColor("#ff79c6"), // pink
		Surface:       themeColor("#44475a"), // current line
	},
	// Nord — https://www.nordtheme.com/docs/colors-and-palettes
	"nord": {
		Accent:        themeColor("#88c0d0"), // nord8 — frost teal
		Muted:         themeColor("#4c566a"), // nord3 — polar night (dim)
		OK:            themeColor("#a3be8c"), // nord14 — aurora green
		Warn:          themeColor("#ebcb8b"), // nord13 — aurora yellow
		Err:           themeColor("#bf616a"), // nord11 — aurora red
		FrameBorder:   themeColor("#3b4252"), // nord1 — polar night
		SearchDim:     themeColor("#4c566a"), // nord3
		RowActiveBG:   themeColor("#434c5e"), // nord2 — selection
		RowActiveFG:   themeColor("#eceff4"), // nord6
		Background:    "#2e3440",             // nord0 — polar night base
		FocusedBorder: themeColor("#88c0d0"), // frost teal (= accent)
		Disabled:      themeColor("#434c5e"), // nord2
		Hint:          themeColor("#4c566a"), // nord3
		Secondary:     themeColor("#81a1c1"), // nord9 — frost light
		Surface:       themeColor("#3b4252"), // nord1
	},
	// Gruvbox dark — https://github.com/morhetz/gruvbox
	"gruvbox": {
		Accent:        themeColor("#fe8019"), // bright orange
		Muted:         themeColor("#928374"), // gray
		OK:            themeColor("#b8bb26"), // bright green
		Warn:          themeColor("#fabd2f"), // bright yellow
		Err:           themeColor("#fb4934"), // bright red
		FrameBorder:   themeColor("#504945"), // bg2 — warm mid-dark
		SearchDim:     themeColor("#928374"), // gray
		RowActiveBG:   themeColor("#665c54"), // bg3 — selection
		RowActiveFG:   themeColor("#ebdbb2"), // fg
		Background:    "#282828",             // bg
		FocusedBorder: themeColor("#fe8019"), // bright orange (= accent)
		Disabled:      themeColor("#504945"), // bg2
		Hint:          themeColor("#665c54"), // bg3
		Secondary:     themeColor("#fabd2f"), // bright yellow
		Surface:       themeColor("#3c3836"), // bg1
	},
	// Catppuccin Mocha — https://github.com/catppuccin/catppuccin
	"catppuccin": {
		Accent:        themeColor("#cba6f7"), // mauve
		Muted:         themeColor("#6c7086"), // overlay0
		OK:            themeColor("#a6e3a1"), // green
		Warn:          themeColor("#f9e2af"), // yellow
		Err:           themeColor("#f38ba8"), // red
		FrameBorder:   themeColor("#45475a"), // surface1 — border
		SearchDim:     themeColor("#585b70"), // overlay2
		RowActiveBG:   themeColor("#585b70"), // overlay2 — selection
		RowActiveFG:   themeColor("#cdd6f4"), // text
		Background:    "#1e1e2e",             // base
		FocusedBorder: themeColor("#cba6f7"), // mauve (= accent)
		Disabled:      themeColor("#313244"), // surface0
		Hint:          themeColor("#6c7086"), // overlay0
		Secondary:     themeColor("#b4befe"), // lavender
		Surface:       themeColor("#313244"), // surface0
	},
	// Kanagawa Wave — https://github.com/rebelot/kanagawa.nvim
	"kanagawa": {
		Accent:        themeColor("#7e9cd8"), // crystalBlue
		Muted:         themeColor("#727169"), // fujiGray
		OK:            themeColor("#98bb6c"), // springGreen
		Warn:          themeColor("#ffa066"), // surimiOrange
		Err:           themeColor("#e46876"), // peachRed
		FrameBorder:   themeColor("#2a2a37"), // bg_p1
		SearchDim:     themeColor("#727169"), // fujiGray
		RowActiveBG:   themeColor("#223249"), // waveBlue1 — selection
		RowActiveFG:   themeColor("#dcd7ba"), // fujiWhite
		Background:    "#1f1f28",             // sumiInk2 / base bg
		FocusedBorder: themeColor("#7e9cd8"), // crystalBlue (= accent)
		Disabled:      themeColor("#363646"), // sumiInk3
		Hint:          themeColor("#54546d"), // sumiInk4
		Secondary:     themeColor("#938aa9"), // springViolet1
		Surface:       themeColor("#2a2a37"), // sumiInk2+
	},
}

var accentPresets = map[string]lipgloss.AdaptiveColor{
	"default": {Light: "25", Dark: "39"},
	"blue":    {Light: "25", Dark: "39"},
	"cyan":    {Light: "30", Dark: "45"},
	"green":   {Light: "28", Dark: "35"},
	"amber":   {Light: "166", Dark: "214"},
	"red":     {Light: "160", Dark: "203"},
	"magenta": {Light: "127", Dark: "213"},
}

// Every style below is (re)built by rebuildAllStyles, which runs on every
// color change and at startup via ApplyColorScheme. Declarations only.
var (
	statusOK   lipgloss.Style
	statusWarn lipgloss.Style
	statusErr  lipgloss.Style
	dim        lipgloss.Style

	// Extended style tokens.
	hintStyle      lipgloss.Style
	secondaryStyle lipgloss.Style
	disabledStyle  lipgloss.Style

	focusedFrameStyle lipgloss.Style

	headerStyle lipgloss.Style
	footerStyle lipgloss.Style

	checkedStyle   lipgloss.Style
	uncheckedStyle lipgloss.Style

	// Active list row: solid background + foreground + bold — no inner styles allowed.
	rowActiveStyle lipgloss.Style

	badgeCfgStyle   lipgloss.Style
	badgeCountStyle lipgloss.Style
	// Selection count pill badge — inverted accent.
	badgeSelStyle lipgloss.Style

	footerKeyStyle lipgloss.Style

	tabActiveStyle   lipgloss.Style
	tabInactiveStyle lipgloss.Style

	searchUnfocused lipgloss.Style

	// Confirm dialog title (referenced in confirm_modal.go).
	confirmTitleStyle lipgloss.Style

	// Help overlay (referenced in help_modal.go).
	helpBoxStyle   lipgloss.Style
	helpTitleStyle lipgloss.Style

	// Toast notifications (referenced in toast.go).
	toastOKStyle   lipgloss.Style
	toastInfoStyle lipgloss.Style
	toastErrStyle  lipgloss.Style
)

func SetAccentColor(name string) {
	// Switching to accent-only mode clears any theme background.
	cBackground = ""
	bgANSICode = ""

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == "default" {
		cAccent = defaultAccent
	} else if v, ok := accentPresets[name]; ok {
		cAccent = v
	} else {
		// Allow arbitrary lipgloss color values ("#RRGGBB", "34", "colour196", ...).
		cAccent = lipgloss.AdaptiveColor{Light: name, Dark: name}
	}

	// Reset extended roles to defaults (no theme).
	cFocusedBorder = cAccent
	cDisabled = lipgloss.AdaptiveColor{Light: "248", Dark: "240"}
	cHint = lipgloss.AdaptiveColor{Light: "245", Dark: "245"}
	cSecondary = lipgloss.AdaptiveColor{Light: "67", Dark: "67"}

	rebuildAllStyles()
}

// ApplyColorScheme applies a named colorscheme if scheme is non-empty and
// known; otherwise falls back to SetAccentColor(accentColor) for backward
// compatibility with configs that only set accent_color.
func ApplyColorScheme(scheme, accentColor string) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if scheme != "" && scheme != "default" {
		if cs, ok := builtinColorSchemes[scheme]; ok {
			applyScheme(cs)
			return
		}
	}
	// scheme is empty, "default", or unknown — fall back to accent logic.
	SetAccentColor(accentColor)
}

// applyScheme mutates all global color vars from cs, then calls rebuildAllStyles.
func applyScheme(cs colorScheme) {
	cAccent = cs.Accent
	cMuted = cs.Muted
	cOK = cs.OK
	cWarn = cs.Warn
	cErr = cs.Err
	cFrameBorder = cs.FrameBorder
	cSearchDim = cs.SearchDim
	cRowActiveBG = cs.RowActiveBG
	cRowActiveFG = cs.RowActiveFG
	cBackground = cs.Background
	bgANSICode = hexToANSIBG(cs.Background)

	// Extended roles.
	cFocusedBorder = cs.FocusedBorder
	cDisabled = cs.Disabled
	cHint = cs.Hint
	cSecondary = cs.Secondary

	rebuildAllStyles()
}

// rebuildAllStyles reconstructs every style variable from the current color
// variables. This is the single source of truth for all style construction.
// Called after any color change (theme switch, accent change).
func rebuildAllStyles() {
	hasBG := cBackground != ""
	var bg lipgloss.TerminalColor
	if hasBG {
		bg = lipgloss.Color(cBackground)
	}

	// Semantic status.
	statusOK = lipgloss.NewStyle().Foreground(cOK)
	statusWarn = lipgloss.NewStyle().Foreground(cWarn)
	statusErr = lipgloss.NewStyle().Foreground(cErr)
	dim = lipgloss.NewStyle().Foreground(cMuted)

	// Extended tokens.
	hintStyle = lipgloss.NewStyle().Foreground(cHint).Italic(true)
	secondaryStyle = lipgloss.NewStyle().Foreground(cSecondary)
	disabledStyle = lipgloss.NewStyle().Foreground(cDisabled).Italic(true)
	underlineFill = lipgloss.NewStyle().Foreground(cFrameBorder)

	// Frame / modal chrome.
	ffs := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(cFocusedBorder).
		Padding(0, 1)
	if hasBG {
		ffs = ffs.Background(bg)
	}
	focusedFrameStyle = ffs

	// Header / footer text.
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	footerStyle = lipgloss.NewStyle().Foreground(cMuted)

	// Selection checkboxes.
	checkedStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	uncheckedStyle = lipgloss.NewStyle().Foreground(cMuted)

	// Active list row highlight.
	rowActiveStyle = lipgloss.NewStyle().Background(cRowActiveBG).Foreground(cRowActiveFG).Bold(true)

	// Badge pills.
	badgeCfgStyle = lipgloss.NewStyle().
		Foreground(cAccent).
		Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"}).
		Padding(0, 1).Bold(true)
	badgeCountStyle = lipgloss.NewStyle().
		Foreground(cMuted).
		Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"}).
		Padding(0, 1)
	badgeSelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "16"}).
		Background(cAccent).
		Padding(0, 1).
		Bold(true)

	// Key hints.
	footerKeyStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)

	// Tab labels.
	tabActiveStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(cMuted)

	// Search bar.
	searchUnfocused = lipgloss.NewStyle().Foreground(cSearchDim)

	// Tab box chrome.
	if hasBG {
		tabBoxBorderStyle = lipgloss.NewStyle().Foreground(cFrameBorder).Background(bg)
		focusedBoxBorderStyle = lipgloss.NewStyle().Foreground(cFocusedBorder).Background(bg)
		tabBoxPadStyle = lipgloss.NewStyle().Background(bg)
	} else {
		tabBoxBorderStyle = lipgloss.NewStyle().Foreground(cFrameBorder)
		focusedBoxBorderStyle = lipgloss.NewStyle().Foreground(cFocusedBorder)
		tabBoxPadStyle = lipgloss.NewStyle()
	}

	// Help overlay.
	hs := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(cFocusedBorder).
		Padding(1, 2)
	if hasBG {
		hs = hs.Background(bg)
	}
	helpBoxStyle = hs
	helpTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)

	// Confirm dialog title.
	confirmTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cErr)

	// Toast notifications.
	toastOKStyle = lipgloss.NewStyle().Foreground(cOK)
	toastInfoStyle = lipgloss.NewStyle().Foreground(cMuted)
	toastErrStyle = lipgloss.NewStyle().Foreground(cErr)
}

// hexToANSIBG converts a "#RRGGBB" hex string to an ANSI truecolor background
// escape sequence "\x1b[48;2;R;G;Bm". Returns "" on parse failure.
func hexToANSIBG(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return ""
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// withThemeBG prepends bgANSICode to s and re-injects it after every ANSI
// reset ("\x1b[0m" or "\x1b[m") so the theme background persists through all
// styled sub-elements. Does nothing when bgANSICode is empty.
func withThemeBG(s string) string {
	if bgANSICode == "" {
		return s
	}
	// Re-inject after every full reset sequence so background is never lost.
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgANSICode)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgANSICode)
	return bgANSICode + s
}

func frameInnerSize(w, h int) (innerW, innerH int) {
	// Frames have a 1-char border on each side + horizontal padding=1.
	innerW = w - 2 - 2
	innerH = h - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	return innerW, innerH
}

func joinHeader(width int, left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if width <= 0 {
		if right == "" {
			return left
		}
		if left == "" {
			return right
		}
		return left + " " + right
	}
	if right == "" {
		return left
	}

	rw := lipgloss.Width(right)
	if rw >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(right)
	}

	if left == "" {
		return strings.Repeat(" ", width-rw) + right
	}

	leftAvail := width - rw - 1
	if leftAvail <= 0 {
		return strings.Repeat(" ", width-rw) + right
	}
	left = lipgloss.NewStyle().MaxWidth(leftAvail).Render(left)
	lw := lipgloss.Width(left)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderFocusedFrame renders a bordered modal frame using focusedFrameStyle
// (cFocusedBorder) so overlays stand out from the background tab box.
func renderFocusedFrame(w, h int, title string, headerRight string, body string, footer string) string {
	if w <= 0 || h <= 0 {
		out := strings.TrimSpace(title)
		if out != "" {
			header := out
			if strings.TrimSpace(headerRight) != "" {
				header = header + " " + strings.TrimSpace(headerRight)
			}
			out = headerStyle.Render(header) + "\n"
		}
		out += strings.TrimSpace(body)
		if strings.TrimSpace(footer) != "" {
			out += "\n" + footer
		}
		return strings.TrimSpace(out)
	}

	innerW, _ := frameInnerSize(w, h)
	head := headerStyle.Render(joinHeader(innerW, title, headerRight))
	foot := ""
	if strings.TrimSpace(footer) != "" {
		foot = footer
	}

	content := strings.TrimRight(head+"\n"+body, "\n")
	if foot != "" {
		content = strings.TrimRight(content, "\n") + "\n" + foot
	}
	box := focusedFrameStyle.Width(w).Height(h).Render(content)
	return withThemeBG(box)
}

func configureSearch(m *textinput.Model) {
	m.PromptStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	m.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "255"})
	m.Cursor.Style = lipgloss.NewStyle().Foreground(cAccent)
	m.PlaceholderStyle = lipgloss.NewStyle().Foreground(cHint).Italic(true)
}

func setSearchFocused(m *textinput.Model, focused bool) {
	if focused {
		configureSearch(m)
		return
	}
	m.PromptStyle = searchUnfocused
	m.TextStyle = searchUnfocused
	m.Cursor.Style = searchUnfocused
}

func setSearchBarFocused(m *textinput.Model, focused bool) {
	setSearchFocused(m, focused)
	if focused {
		m.Placeholder = "search"
	} else {
		m.Placeholder = "type to search..."
	}
}

// styledFooter renders a footer string with keys in accent and actions dimmed.
// Input format: "⏎ connect  ␣ select  o panes" (double-space separated hints).
func styledFooter(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "  ")
		styled := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if p == "·" {
				styled = append(styled, hintStyle.Render("·"))
				continue
			}
			idx := strings.IndexByte(p, ' ')
			if idx < 0 {
				styled = append(styled, footerKeyStyle.Render(p))
				continue
			}
			k := p[:idx]
			a := p[idx:] // includes leading space
			styled = append(styled, footerKeyStyle.Render(k)+hintStyle.Render(a))
		}
		out = append(out, strings.Join(styled, "  "))
	}
	return strings.Join(out, "\n")
}

// Spinner for loading states.
var (
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIndex  int
	spinnerActive bool
	// spinnerMinEnd ensures the spinner shows for at least spinnerMinDuration.
	spinnerMinEnd time.Time
)

const (
	spinnerTickInterval = 80 * time.Millisecond
	spinnerMinDuration  = 600 * time.Millisecond
)

type spinnerTickMsg struct{}

// spinnerStart activates the spinner with a minimum visible duration.
func spinnerStart() {
	spinnerActive = true
	spinnerIndex = 0
	spinnerMinEnd = time.Now().Add(spinnerMinDuration)
}

// spinnerStop deactivates the spinner (but it may keep running until min duration).
func spinnerStop() {
	// The tick handler checks time and will deactivate.
	if time.Now().After(spinnerMinEnd) {
		spinnerActive = false
	}
	// Otherwise the tick handler will stop it when minEnd is reached.
}

func spinnerFrame() string {
	return spinnerFrames[spinnerIndex%len(spinnerFrames)]
}

// formScrollWindow returns the slice of lines to display, scrolled so that
// focusLine is visible within visibleH lines. Returns (start, end) indices.
func formScrollWindow(totalLines, visibleH, focusLine int) (int, int) {
	if totalLines <= visibleH {
		return 0, totalLines
	}
	// Center focused line in the window.
	start := focusLine - visibleH/2
	if start < 0 {
		start = 0
	}
	end := start + visibleH
	if end > totalLines {
		end = totalLines
		start = end - visibleH
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// formSection renders a lightweight section divider: "── Label ──────"
func formSection(label string, width int) string {
	label = strings.TrimSpace(label)
	seg := "── " + label + " "
	segW := lipgloss.Width(seg)
	fill := width - segW
	if fill < 0 {
		fill = 0
	}
	return secondaryStyle.Render(seg + strings.Repeat("─", fill))
}

// statusDot returns a colored dot for status display.
func statusDot(ok bool, hasWarnings bool) string {
	if !ok || hasWarnings {
		return statusWarn.Render("●")
	}
	return statusOK.Render("●")
}
