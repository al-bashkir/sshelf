package ui

import (
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type groupFormCancelMsg struct{}

type groupFormSaveMsg struct {
	index int
	group config.Group
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func groupSchema(defs config.Defaults) formSchema {
	portPlaceholder, identPlaceholder, extraPlaceholder := sshPlaceholders(defs)

	return formSchema{
		Sections: []sectionDef{
			{Key: "group", Label: "Group", Fields: []fieldDef{
				{Key: "name", Label: "Name", Kind: fieldText,
					Placeholder: "prod", CharLimit: 128},
			}},
			{Key: "ssh", Label: "SSH", Fields: []fieldDef{
				{Key: "user", Label: "User", Kind: fieldText,
					Placeholder: strings.TrimSpace(defs.User), CharLimit: 128},
				{Key: "port", Label: "Port", Kind: fieldNumber,
					Placeholder: portPlaceholder, CharLimit: 16, Narrow: true,
					Validate: validatePort},
				{Key: "identity", Label: "Identity file", Kind: fieldText,
					Placeholder: identPlaceholder, CharLimit: 512},
				{Key: "extra_args", Label: "Extra args", Kind: fieldText,
					Placeholder: extraPlaceholder, CharLimit: 1024},
				{Key: "remote_cmd", Label: "Remote cmd", Kind: fieldText,
					Placeholder: "command to run on connect", CharLimit: 1024},
			}},
			{Key: "tmux", Label: "Tmux", Fields: []fieldDef{
				{Key: "open_mode", Label: "Open mode", Kind: fieldPicker,
					Options: optsInherit("auto", "current", "tmux-window", "tmux-pane")},
				{Key: "tmux", Label: "Tmux", Kind: fieldPicker,
					Options: optsInherit("auto", "force", "never")},
			}},
			{Key: "panes", Label: "Panes", Fields: []fieldDef{
				{Key: "pane_split", Label: "Pane split", Kind: fieldPicker,
					Options: optsInherit("horizontal", "vertical")},
				{Key: "pane_layout", Label: "Pane layout", Kind: fieldPicker,
					Options: optsInherit("auto", "tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical")},
				{Key: "pane_sync", Label: "Pane sync", Kind: fieldPicker,
					Options: optsInherit("on", "off")},
				{Key: "border_pos", Label: "Pane border", Kind: fieldPicker,
					Options: optsInherit("bottom", "top", "off")},
				{Key: "border_fmt", Label: "Border format", Kind: fieldSubModal},
			}},
		},
	}
}

// ---------------------------------------------------------------------------
// Value conversion
// ---------------------------------------------------------------------------

func groupToValues(g config.Group) map[string]string {
	portStr := ""
	if g.Port != 0 {
		portStr = strconv.Itoa(g.Port)
	}
	return map[string]string{
		"name":        strings.TrimSpace(g.Name),
		"user":        strings.TrimSpace(g.User),
		"port":        portStr,
		"identity":    strings.TrimSpace(g.IdentityFile),
		"extra_args":  strings.Join(g.ExtraArgs, " "),
		"remote_cmd":  strings.TrimSpace(g.RemoteCommand),
		"open_mode":   g.OpenMode,
		"tmux":        g.Tmux,
		"pane_split":  g.PaneSplit,
		"pane_layout": g.PaneLayout,
		"pane_sync":   g.PaneSync,
		"border_pos":  g.PaneBorderPos,
		"border_fmt":  g.PaneBorderFmt,
	}
}

func applyGroupValues(values map[string]string, base config.Group) (config.Group, error) {
	g := base
	g.Name = strings.TrimSpace(values["name"])
	g.User = strings.TrimSpace(values["user"])
	g.IdentityFile = strings.TrimSpace(values["identity"])
	g.RemoteCommand = strings.TrimSpace(values["remote_cmd"])
	g.PaneBorderFmt = strings.TrimSpace(values["border_fmt"])
	g.OpenMode = values["open_mode"]
	g.Tmux = values["tmux"]
	g.PaneSplit = values["pane_split"]
	g.PaneLayout = values["pane_layout"]
	g.PaneSync = values["pane_sync"]
	g.PaneBorderPos = values["border_pos"]

	p, err := parsePort(values["port"])
	if err != nil {
		return g, err
	}
	g.Port = p

	g.ExtraArgs = parseExtraArgs(values["extra_args"], base.ExtraArgs)

	return g, config.ValidateGroupName(g.Name)
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type groupFormModel struct {
	form        formModel
	index       int
	group       config.Group
	defs        config.Defaults
	parentCrumb string

	width  int
	height int

	borderPicker *paneBorderFormatsModel

	toast toast

	keymap keyMap
}

func (m *groupFormModel) refreshAccentStyles() {
	m.form.applyFocusStyles()
	if m.borderPicker != nil {
		m.borderPicker.refreshAccentStyles()
	}
}

func newGroupFormModel(index int, g config.Group, defs config.Defaults) *groupFormModel {
	// Reasonable default for new groups.
	if index < 0 && strings.TrimSpace(g.OpenMode) == "" {
		g.OpenMode = "tmux-window"
	}

	values := groupToValues(g)
	fm := newFormModel(groupSchema(defs), values, modalFormLabelWidth, defs.ShowFieldHelp)

	return &groupFormModel{
		form:   fm,
		index:  index,
		group:  g,
		defs:   defs,
		keymap: defaultKeyMap(),
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *groupFormModel) Init() tea.Cmd { return nil }

func (m *groupFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case paneBorderFormatsCancelMsg:
		m.borderPicker = nil
		return m, nil
	case paneBorderFormatsDoneMsg:
		m.group.PaneBorderFmt = strings.TrimSpace(msg.value)
		m.form.setValue("border_fmt", m.group.PaneBorderFmt)
		m.borderPicker = nil
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.form.handleResize(msg.Width, msg.Height)
		if m.borderPicker != nil {
			mw, mh := pickerModalSize(msg.Width, msg.Height)
			_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		return m, nil

	case tea.KeyMsg:
		if m.borderPicker != nil {
			model, cmd := m.borderPicker.Update(msg)
			if pm, ok := model.(*paneBorderFormatsModel); ok {
				m.borderPicker = pm
			}
			return m, cmd
		}

		if m.form.askingDiscard() {
			return m, m.form.handleDiscardKey(msg, &m.toast)
		}

		// In insert mode or picker popup, delegate to form first.
		if m.form.editing || m.form.picker != nil {
			handled, cmd := m.form.handleKey(msg)
			if handled {
				return m, cmd
			}
		}

		// Save.
		if key.Matches(msg, m.keymap.Save) {
			m.form.exitEdit()
			m.toast = toast{}
			g, err := applyGroupValues(m.form.values, m.group)
			if err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return groupFormSaveMsg{index: m.index, group: g} }
		}

		if key.Matches(msg, m.keymap.Esc) {
			return m, m.form.exitOrAsk(func() tea.Msg { return groupFormCancelMsg{} }, &m.toast)
		}

		// Sub-modal activation for border format.
		fk := m.form.focusedFieldKey()
		s := msg.String()
		if fk == "border_fmt" && (s == "enter" || s == " " || s == "l") {
			mw, mh := pickerModalSize(m.width, m.height)
			m.borderPicker = newPaneBorderFormatsModel(m.defs, m.group.PaneBorderFmt, true, false)
			m.borderPicker.parentCrumb = m.parentCrumb
			if mw > 0 && mh > 0 {
				_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			}
			return m, nil
		}

		// Generic form key handling.
		handled, cmd := m.form.handleKey(msg)
		if handled {
			return m, cmd
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *groupFormModel) View() string {
	if m.borderPicker != nil {
		return placeCentered(m.width, m.height, m.borderPicker.View())
	}

	name := m.group.Name
	if strings.TrimSpace(name) == "" {
		name = m.form.value("name")
	}
	title := formTitle(m.parentCrumb, m.index, name, "Create Group", "Edit Group")
	return renderModalForm(&m.form, m.width, m.height, title, m.toast)
}
