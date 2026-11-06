package ui

import (
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

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

// groupSchema is the group form; parents are the choices for its parent.
func groupSchema(defs config.Defaults, parents []fieldOption) formSchema {
	portPlaceholder, identPlaceholder, extraPlaceholder := sshPlaceholders(defs)

	return formSchema{
		Sections: []sectionDef{
			{Key: "group", Label: "Group", Fields: []fieldDef{
				{Key: "name", Label: "Name", Kind: fieldText,
					Placeholder: "prod", CharLimit: 128},
				{Key: "parent", Label: "Parent", Kind: fieldPicker, Options: parents},
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
		"parent":      g.Parent,
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
	g.Parent = values["parent"]
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
	formScreen
	index int
	group config.Group
	defs  config.Defaults // the custom border formats to choose from
}

// newGroupFormModel opens the form of group index (-1: a new group) in groups.
func newGroupFormModel(index int, g config.Group, groups []config.Group, defs config.Defaults) *groupFormModel {
	// Reasonable default for new groups.
	if index < 0 && strings.TrimSpace(g.OpenMode) == "" {
		g.OpenMode = "tmux-window"
	}

	values := groupToValues(g)
	fm := newFormModel(groupSchema(defs, parentOptions(groups, index)), values, modalFormLabelWidth, defs.ShowFieldHelp)

	return &groupFormModel{
		formScreen: newFormScreen(fm),
		index:      index,
		group:      g,
		defs:       defs,
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *groupFormModel) Init() tea.Cmd { return nil }

func (m *groupFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.update(msg, formHooks{save: m.saveMsg, cancel: groupFormCancelMsg{}, borders: m.borders})
}

func (m *groupFormModel) saveMsg() (tea.Msg, error) {
	g, err := applyGroupValues(m.form.values, m.group)
	return groupFormSaveMsg{index: m.index, group: g}, err
}

// borders opens the formats from the defaults, with an inherit choice. A
// group cannot add or delete formats.
func (m *groupFormModel) borders() *paneBorderFormatsModel {
	return newPaneBorderFormatsModel(m.defs, m.form.value("border_fmt"), true, false)
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

// parentOptions lists "none" and, in tree order, the groups that group
// index can sit in.
func parentOptions(groups []config.Group, index int) []fieldOption {
	out := []fieldOption{{Value: "", Display: "none"}}
	for _, n := range config.ParentChoices(groups, index) {
		out = append(out, fieldOption{Value: groups[n.Index].Name, Display: groups[n.Index].Name})
	}
	return out
}
