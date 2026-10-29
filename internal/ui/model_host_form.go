package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func hostSchema(defs config.Defaults) formSchema {
	portPlaceholder, identPlaceholder, extraPlaceholder := sshPlaceholders(defs)

	return formSchema{
		Sections: []sectionDef{
			{Key: "conn", Label: "Connection", Fields: []fieldDef{
				{Key: "host", Label: "Host", Kind: fieldText,
					Placeholder: "example.com or [10.0.0.1]:2222", CharLimit: 512},
				{Key: "user", Label: "User", Kind: fieldText,
					Placeholder: strings.TrimSpace(defs.User), CharLimit: 128},
				{Key: "port", Label: "Port", Kind: fieldNumber,
					Placeholder: portPlaceholder, CharLimit: 16, Narrow: true,
					Validate: validatePort},
			}},
			{Key: "auth", Label: "Authentication", Fields: []fieldDef{
				{Key: "identity", Label: "Identity file", Kind: fieldText,
					Placeholder: identPlaceholder, CharLimit: 512},
				{Key: "extra_args", Label: "Extra args", Kind: fieldText,
					Placeholder: extraPlaceholder, CharLimit: 1024},
			}},
		},
	}
}

// ---------------------------------------------------------------------------
// Value conversion
// ---------------------------------------------------------------------------

func hostToValues(h config.Host) map[string]string {
	portStr := ""
	if h.Port != 0 {
		portStr = strconv.Itoa(h.Port)
	}
	return map[string]string{
		"host":       strings.TrimSpace(h.Host),
		"user":       strings.TrimSpace(h.User),
		"port":       portStr,
		"identity":   strings.TrimSpace(h.IdentityFile),
		"extra_args": strings.Join(h.ExtraArgs, " "),
	}
}

func applyHostValues(values map[string]string, base config.Host) (config.Host, error) {
	h := base
	h.Host = strings.TrimSpace(values["host"])
	h.User = strings.TrimSpace(values["user"])
	h.IdentityFile = strings.TrimSpace(values["identity"])

	p, err := parsePort(values["port"])
	if err != nil {
		return h, err
	}
	h.Port = p

	h.ExtraArgs = parseExtraArgs(values["extra_args"], base.ExtraArgs)

	if strings.TrimSpace(h.Host) == "" {
		return h, fmt.Errorf("host required")
	}
	return h, nil
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type hostFormModel struct {
	form        formModel
	index       int
	host        config.Host
	parentCrumb string

	width  int
	height int

	toast toast

	keymap keyMap
}

func (m *hostFormModel) refreshAccentStyles() {
	m.form.applyFocusStyles()
}

func newHostFormModel(index int, h config.Host, defs config.Defaults) *hostFormModel {
	values := hostToValues(h)
	fm := newFormModel(hostSchema(defs), values, modalFormLabelWidth, defs.ShowFieldHelp)

	return &hostFormModel{
		form:   fm,
		index:  index,
		host:   h,
		keymap: defaultKeyMap(),
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *hostFormModel) Init() tea.Cmd { return nil }

func (m *hostFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.form.handleResize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if m.form.askingDiscard() {
			return m, m.form.handleDiscardKey(msg, &m.toast)
		}

		// Insert mode — delegate to form first.
		if m.form.editing {
			handled, cmd := m.form.handleKey(msg)
			if handled {
				return m, cmd
			}
		}

		// Save.
		if key.Matches(msg, m.keymap.Save) {
			m.form.exitEdit()
			m.toast = toast{}
			h, err := applyHostValues(m.form.values, m.host)
			if err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return hostFormSaveMsg{index: m.index, host: h} }
		}

		if key.Matches(msg, m.keymap.Esc) {
			return m, m.form.exitOrAsk(func() tea.Msg { return hostFormCancelMsg{} }, &m.toast)
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

func (m *hostFormModel) View() string {
	name := m.host.Host
	if strings.TrimSpace(name) == "" {
		name = m.form.value("host")
	}
	title := formTitle(m.parentCrumb, m.index, name, "Create Host", "Edit Host")
	return renderModalForm(&m.form, m.width, m.height, title, m.toast)
}
