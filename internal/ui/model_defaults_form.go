package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type defaultsFormCancelMsg struct{}

type defaultsFormSaveMsg struct {
	defaults config.Defaults
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// settingsSchema returns the declarative form definition for the Settings
// screen, organized into five sections following Variant A layout.
func settingsSchema() formSchema {
	return formSchema{
		Sections: []sectionDef{
			{Key: "ssh", Label: "SSH", Fields: []fieldDef{
				{Key: "user", Label: "User", Kind: fieldText,
					Placeholder: "login username",
					Helper:      "Default SSH username for all connections",
					CharLimit:   128},
				{Key: "port", Label: "Port", Kind: fieldNumber,
					Default: "22", Placeholder: "22",
					Helper:    "Default SSH port (1\u201365535)",
					CharLimit: 6, Narrow: true,
					Validate: validatePort},
				{Key: "identity", Label: "Identity file", Kind: fieldText,
					Placeholder: "~/.ssh/id_ed25519",
					Helper:      "Path to SSH private key for authentication",
					CharLimit:   512},
				{Key: "extra_args", Label: "Extra arguments", Kind: fieldText,
					Placeholder: "-o Option=value ...",
					Helper:      "Additional ssh flags, space-separated",
					CharLimit:   1024},
				{Key: "load_known_hosts", Label: "Load known_hosts", Kind: fieldPicker,
					Options: opts("yes", "no"), Default: "yes",
					Helper: "Auto-import hosts from ~/.ssh/known_hosts"},
			}},
			{Key: "appearance", Label: "Appearance", Fields: []fieldDef{
				{Key: "colorscheme", Label: "Colorscheme", Kind: fieldPicker,
					Options: opts("default", "dracula", "nord", "gruvbox", "catppuccin", "kanagawa"),
					Helper:  "Application color theme"},
				{Key: "accent", Label: "Accent color", Kind: fieldPicker,
					Options: opts("default", "blue", "cyan", "green", "amber", "red", "magenta"),
					Helper:  "UI accent color (overridden when a colorscheme is active)",
					DisabledWhen: func(values map[string]string) string {
						if cs := values["colorscheme"]; cs != "" && cs != "default" {
							return "overridden by " + cs + " theme"
						}
						return ""
					}},
				{Key: "show_field_help", Label: "Field help", Kind: fieldPicker,
					Options: opts("yes", "no"), Default: "yes",
					Helper: "Show helper text below focused form fields"},
			}},
			{Key: "tmux", Label: "Tmux", Fields: []fieldDef{
				{Key: "tmux", Label: "Tmux mode", Kind: fieldPicker,
					Options: opts("auto", "force", "never"),
					Helper:  "auto: use tmux when available, force: require, never: disable"},
				{Key: "open_mode", Label: "Open mode", Kind: fieldPicker,
					Options: opts("auto", "current", "tmux-window", "tmux-pane"),
					Helper:  "How to open new SSH sessions"},
				{Key: "session", Label: "Session name", Kind: fieldText,
					Default: "sshelf", Placeholder: "sshelf",
					Helper:    "Name of the tmux session to use",
					CharLimit: 128},
			}},
			{Key: "behavior", Label: "Behavior", Fields: []fieldDef{
				{Key: "confirm_quit", Label: "Confirm quit", Kind: fieldPicker,
					Options: opts("yes", "no"), Default: "no",
					Helper: "Show confirmation dialog before quitting"},
				{Key: "threshold", Label: "Confirm threshold", Kind: fieldNumber,
					Default: "5", Placeholder: "5",
					Helper:    "Ask before connecting to more than N hosts (0 = never)",
					CharLimit: 6, Narrow: true,
					Validate: validateNonNegativeInt},
			}},
			{Key: "panes", Label: "Panes", Fields: []fieldDef{
				{Key: "pane_split", Label: "Split direction", Kind: fieldPicker,
					Options: opts("horizontal", "vertical"),
					Helper:  "Direction for splitting tmux panes"},
				{Key: "pane_layout", Label: "Layout", Kind: fieldPicker,
					Options: opts("auto", "tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical"),
					Helper:  "Tmux pane arrangement algorithm"},
				{Key: "pane_sync", Label: "Synchronize input", Kind: fieldPicker,
					Options: opts("on", "off"), Default: "on",
					Helper: "Send keystrokes to all panes simultaneously"},
				{Key: "border_pos", Label: "Border position", Kind: fieldPicker,
					Options: opts("bottom", "top", "off"),
					Helper:  "Position of pane border status line"},
				{Key: "border_fmt", Label: "Border format", Kind: fieldSubModal,
					Helper: "Tmux format string for pane borders"},
			}},
		},
	}
}

// ---------------------------------------------------------------------------
// Value conversion: config.Defaults <-> map[string]string
// ---------------------------------------------------------------------------

// defaultsToValues maps a config.Defaults to the flat string map used by
// the form model.
func defaultsToValues(d config.Defaults) map[string]string {
	loadKH := "no"
	if d.LoadKnownHosts {
		loadKH = "yes"
	}
	confirmQuit := "no"
	if d.ConfirmQuit {
		confirmQuit = "yes"
	}
	showFieldHelp := "no"
	if d.ShowFieldHelp {
		showFieldHelp = "yes"
	}
	portStr := ""
	if d.Port != 0 {
		portStr = strconv.Itoa(d.Port)
	}
	threshStr := ""
	if d.ConnectConfirmThreshold >= 0 {
		threshStr = strconv.Itoa(d.ConnectConfirmThreshold)
	}
	return map[string]string{
		"user":             strings.TrimSpace(d.User),
		"port":             portStr,
		"identity":         strings.TrimSpace(d.IdentityFile),
		"extra_args":       strings.Join(d.ExtraArgs, " "),
		"colorscheme":      d.Colorscheme,
		"accent":           d.AccentColor,
		"load_known_hosts": loadKH,
		"tmux":             d.Tmux,
		"open_mode":        d.OpenMode,
		"session":          strings.TrimSpace(d.TmuxSession),
		"confirm_quit":     confirmQuit,
		"show_field_help":  showFieldHelp,
		"threshold":        threshStr,
		"pane_split":       d.PaneSplit,
		"pane_layout":      d.PaneLayout,
		"pane_sync":        d.PaneSync,
		"border_pos":       d.PaneBorderPos,
		"border_fmt":       d.PaneBorderFmt,
	}
}

// applySettingsValues converts form values back to a config.Defaults.
func applySettingsValues(values map[string]string, base config.Defaults) (config.Defaults, error) {
	d := base

	d.User = strings.TrimSpace(values["user"])
	d.IdentityFile = strings.TrimSpace(values["identity"])

	// Colorscheme / accent.
	d.Colorscheme = strings.ToLower(strings.TrimSpace(values["colorscheme"]))
	if d.Colorscheme == "default" {
		d.Colorscheme = ""
	}
	d.AccentColor = strings.ToLower(strings.TrimSpace(values["accent"]))
	if d.AccentColor == "default" {
		d.AccentColor = ""
	}

	d.LoadKnownHosts = values["load_known_hosts"] != "no"
	d.ConfirmQuit = values["confirm_quit"] == "yes"
	d.ShowFieldHelp = values["show_field_help"] != "no"

	// Port.
	p, err := parsePort(values["port"])
	if err != nil {
		return d, err
	}
	d.Port = p
	if p == 0 {
		d.Port = 22
	}

	// Extra args.
	d.ExtraArgs = parseExtraArgs(values["extra_args"], base.ExtraArgs)

	// Tmux fields.
	d.Tmux = strings.TrimSpace(values["tmux"])
	d.OpenMode = strings.TrimSpace(values["open_mode"])
	d.TmuxSession = strings.TrimSpace(values["session"])
	if d.TmuxSession == "" {
		d.TmuxSession = "sshelf"
	}

	t, err := parseThreshold(values["threshold"])
	if err != nil {
		return d, err
	}
	d.ConnectConfirmThreshold = t

	// Pane fields.
	d.PaneSplit = strings.TrimSpace(values["pane_split"])
	d.PaneLayout = strings.TrimSpace(values["pane_layout"])
	d.PaneSync = strings.TrimSpace(values["pane_sync"])
	d.PaneBorderPos = strings.TrimSpace(values["border_pos"])

	// Border format (may have been updated via sub-modal).
	d.PaneBorderFmt = strings.TrimSpace(values["border_fmt"])
	if d.PaneBorderFmt == "" {
		d.PaneBorderFmt = config.DefaultPaneBorderFormat
	}

	d.PaneBorderFmts = cleanBorderFormats(d.PaneBorderFmts)
	return d, nil
}

// parseThreshold parses the connect-confirm threshold; empty means 5.
func parseThreshold(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 5, nil
	}
	t, err := strconv.Atoi(s)
	if err != nil || t < 0 {
		return 0, fmt.Errorf("confirm threshold must be a number >= 0")
	}
	return t, nil
}

// cleanBorderFormats trims the custom pane border formats and drops blanks,
// duplicates and the built-in default.
func cleanBorderFormats(fmts []string) []string {
	clean := make([]string, 0, len(fmts))
	seen := map[string]bool{strings.TrimSpace(config.DefaultPaneBorderFormat): true}
	for _, s := range fmts {
		v := strings.TrimSpace(s)
		if v == "" || seen[v] {
			continue
		}
		clean = append(clean, v)
		seen[v] = true
	}
	return clean
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type defaultsFormModel struct {
	formScreen
	defaults config.Defaults // the base for save, with the custom border formats

	confirmQuitEnabled bool
	confirmQuit        bool
}

func newDefaultsFormModel(d config.Defaults, confirmQuitEnabled bool) *defaultsFormModel {
	values := defaultsToValues(d)
	fm := newFormModel(settingsSchema(), values, defaultsFormLabelWidth, d.ShowFieldHelp)

	s := newFormScreen(fm)
	s.parentCrumb = "Settings"
	return &defaultsFormModel{
		formScreen:         s,
		defaults:           d,
		confirmQuitEnabled: confirmQuitEnabled,
	}
}

func (m *defaultsFormModel) Init() tea.Cmd { return nil }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *defaultsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(paneBorderFormatsDoneMsg); ok {
		m.defaults.PaneBorderFmts = append([]string(nil), done.customFmts...)
	}
	cmd := m.update(msg, formHooks{
		save: m.saveMsg, cancel: defaultsFormCancelMsg{}, ownKey: m.ownKey, borders: m.borders,
	})
	// Live preview of the field help choice.
	m.form.showFieldHelp = m.form.value("show_field_help") != "no"
	return m, cmd
}

func (m *defaultsFormModel) saveMsg() (tea.Msg, error) {
	d, err := applySettingsValues(m.form.values, m.defaults)
	return defaultsFormSaveMsg{defaults: d}, err
}

// borders opens the formats with no inherit choice. Settings can add and
// delete formats.
func (m *defaultsFormModel) borders() *paneBorderFormatsModel {
	return newPaneBorderFormatsModel(m.defaults, m.form.value("border_fmt"), false, true)
}

// ownKey handles the quit question, the tab keys and q.
func (m *defaultsFormModel) ownKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if handled, cmd := handleConfirmQuit(msg, &m.confirmQuit, &m.toast, nil, true); handled {
		return cmd, true
	}
	switch {
	case key.Matches(msg, m.keymap.HostsTab):
		return m.form.exitOrAsk(msgCmd(switchScreenMsg{to: screenHosts}), &m.toast), true
	case key.Matches(msg, m.keymap.GroupsTab):
		return m.form.exitOrAsk(msgCmd(switchScreenMsg{to: screenGroups}), &m.toast), true
	case key.Matches(msg, m.keymap.Quit):
		return m.quit(), true
	}
	return nil, false
}

// quit asks first when confirm-quit is on; otherwise only unsaved changes
// stop it.
func (m *defaultsFormModel) quit() tea.Cmd {
	if m.confirmQuitEnabled {
		return askQuit(true, &m.confirmQuit, &m.toast, nil)
	}
	return m.form.exitOrAsk(tea.Quit, &m.toast)
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *defaultsFormModel) View() string {
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.borderPicker != nil {
		return placeCentered(m.width, m.height, m.borderPicker.View())
	}

	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	contentH := max(0, innerH-tabBoxHeaderLines)

	// Build content lines from the form model.
	lines, focusLine := m.form.renderFormContent(innerW)

	// Footer.
	footer := m.form.renderFooter()

	// Toast.
	toastStr := ""
	if !m.toast.empty() {
		toastStr = renderToast(m.toast)
	}

	// Reserve lines for toast + footer.
	reservedBottom := 1 // footer
	if toastStr != "" {
		reservedBottom++
	}
	visibleH := max(1, contentH-reservedBottom) // keep the focused row

	// Scroll so focused field is visible.
	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]

	// Overlay picker popup as a dropdown below the focused field.
	if m.form.picker != nil {
		focusRow := focusLine - start
		visible = m.form.overlayPickerOnVisible(visible, focusRow, innerW)
	}

	contentLines := make([]string, 0, contentH)
	contentLines = append(contentLines, visible...)
	// Fill remaining space.
	for i := len(visible); i < visibleH; i++ {
		contentLines = append(contentLines, "")
	}
	if toastStr != "" {
		contentLines = append(contentLines, toastStr)
	}
	contentLines = append(contentLines, footer)

	// Header.
	headLeft := headerStyle.Render("Settings")
	headRight := ""
	if m.form.isDirty() {
		headRight = lipgloss.NewStyle().Foreground(cWarn).Render("[mod]") + " "
	}
	headRight += statusDot(false)

	return renderMainTabBox(m.width, m.height, 2, headLeft, headRight, strings.Join(contentLines, "\n"))
}
