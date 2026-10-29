package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHosts screen = iota
	screenGroups
	screenGroupForm
	screenGroupHosts
	screenHostPicker
	screenGroupPicker
	screenDefaultsForm
	screenCustomHost
	screenHostForm
)

type switchScreenMsg struct {
	to screen
}

type openGroupFormMsg struct {
	index int
}

type openGroupFormPrefillMsg struct {
	group config.Group
}

type deleteGroupMsg struct {
	index int
}

type openGroupHostsMsg struct {
	index int
}

type openHostPickerMsg struct {
	groupIndex int
	returnTo   screen
}

type openGroupPickerMsg struct {
	hosts []string
}

type openDefaultsFormMsg struct {
	returnTo screen
}

type openCustomHostMsg struct {
	returnTo   screen
	groupIndex int // -1 means no group context; Ctrl+a will pick a group
}

type openHostFormMsg struct {
	host     string
	returnTo screen
}

type openHostFormPrefillMsg struct {
	host     config.Host
	returnTo screen
}

type customHostCancelMsg struct{}

type customHostConnectMsg struct {
	returnTo   screen
	groupIndex int
	hosts      []string
}

type customHostDoneMsg struct {
	returnTo   screen
	groupIndex int
	hosts      []string
}

type customHostPickGroupMsg struct {
	returnTo screen
	hosts    []string
}

type groupPickerCancelMsg struct{}

type groupPickerDoneMsg struct {
	groupIndex int
}

type removeHostsMsg struct {
	groupIndex int
	hosts      []string
}

type hostFormCancelMsg struct{}

type hostFormSaveMsg struct {
	index int
	host  config.Host
}

type toggleHiddenHostMsg struct {
	host string
	hide bool
}

type defaultsToastExpireMsg struct {
	token int
}

type toastDismissMsg struct {
	token int
}

type appModel struct {
	opts Options

	width  int
	height int

	screen             screen
	hosts              *hostsModel
	groups             *groupsModel
	form               *groupFormModel
	gh                 *groupHostsModel
	picker             *hostPickerModel
	gp                 *groupPickerModel
	defaultsForm       *defaultsFormModel
	customHost         *customHostModel
	hostForm           *hostFormModel
	gpHosts            []string
	gpReturnTo         screen
	returnTo           screen
	returnGroupIndex   int
	defaultsReturnTo   screen
	hostFormReturnTo   screen
	defaultsToastToken int
	toastToken         int

	quitting bool
	execCmd  []string
}

func newAppModel(opts Options) *appModel {
	m := &appModel{
		opts:   opts,
		screen: screenHosts,
		hosts:  newHostsModel(opts),
		groups: newGroupsModel(opts),
	}
	return m
}

func (m *appModel) Init() tea.Cmd {
	return m.hosts.Init()
}

func (m *appModel) applyWindowSize(ws tea.WindowSizeMsg) tea.Cmd {
	m.width = ws.Width
	m.height = ws.Height

	var cmds []tea.Cmd
	if m.hosts != nil {
		model, cmd := m.hosts.Update(ws)
		if hm, ok := model.(*hostsModel); ok {
			m.hosts = hm
		}
		cmds = append(cmds, cmd)
	}
	if m.groups != nil {
		model, cmd := m.groups.Update(ws)
		if gm, ok := model.(*groupsModel); ok {
			m.groups = gm
		}
		cmds = append(cmds, cmd)
	}
	if m.form != nil {
		mw, mh := groupFormModalSize(ws.Width, ws.Height)
		model, cmd := m.form.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if fm, ok := model.(*groupFormModel); ok {
			m.form = fm
		}
		cmds = append(cmds, cmd)
	}
	if m.gh != nil {
		model, cmd := m.gh.Update(ws)
		if gh, ok := model.(*groupHostsModel); ok {
			m.gh = gh
		}
		cmds = append(cmds, cmd)
	}
	if m.picker != nil {
		mw, mh := pickerModalSize(ws.Width, ws.Height)
		model, cmd := m.picker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if pm, ok := model.(*hostPickerModel); ok {
			m.picker = pm
		}
		cmds = append(cmds, cmd)
	}
	if m.gp != nil {
		mw, mh := pickerModalSize(ws.Width, ws.Height)
		model, cmd := m.gp.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if gm, ok := model.(*groupPickerModel); ok {
			m.gp = gm
		}
		cmds = append(cmds, cmd)
	}
	if m.defaultsForm != nil {
		model, cmd := m.defaultsForm.Update(ws)
		if dm, ok := model.(*defaultsFormModel); ok {
			m.defaultsForm = dm
		}
		cmds = append(cmds, cmd)
	}
	if m.customHost != nil {
		mw, mh := customHostModalSize(ws.Width, ws.Height)
		model, cmd := m.customHost.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if cm, ok := model.(*customHostModel); ok {
			m.customHost = cm
		}
		cmds = append(cmds, cmd)
	}
	if m.hostForm != nil {
		mw, mh := hostFormModalSize(ws.Width, ws.Height)
		model, cmd := m.hostForm.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if hm, ok := model.(*hostFormModel); ok {
			m.hostForm = hm
		}
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *appModel) collectToastKey() string {
	var b strings.Builder
	if m.hosts != nil && !m.hosts.toast.empty() {
		b.WriteString(m.hosts.toast.text)
	}
	if m.groups != nil && !m.groups.toast.empty() {
		b.WriteByte('|')
		b.WriteString(m.groups.toast.text)
	}
	if m.gh != nil && !m.gh.toast.empty() {
		b.WriteByte('|')
		b.WriteString(m.gh.toast.text)
	}
	return b.String()
}

func (m *appModel) clearToasts() {
	if m.hosts != nil {
		m.hosts.toast = toast{}
	}
	if m.groups != nil {
		m.groups.toast = toast{}
	}
	if m.gh != nil {
		m.gh.toast = toast{}
	}
}

func (m *appModel) maxToastLevel() toastLevel {
	var lvl toastLevel
	if m.hosts != nil && !m.hosts.toast.empty() && m.hosts.toast.level > lvl {
		lvl = m.hosts.toast.level
	}
	if m.groups != nil && !m.groups.toast.empty() && m.groups.toast.level > lvl {
		lvl = m.groups.toast.level
	}
	if m.gh != nil && !m.gh.toast.empty() && m.gh.toast.level > lvl {
		lvl = m.gh.toast.level
	}
	return lvl
}

func (m *appModel) breadcrumb() string {
	switch m.screen {
	case screenHosts:
		return "Hosts"
	case screenGroups:
		return "Groups"
	case screenGroupHosts:
		if m.gh != nil {
			return "Groups > " + m.gh.group.Name
		}
		return "Groups"
	case screenDefaultsForm:
		return "Settings"
	default:
		return ""
	}
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Spinner tick.
	if _, ok := msg.(spinnerTickMsg); ok {
		if spinnerActive {
			spinnerIndex++
			// Check if minimum duration has elapsed and work is done.
			if !spinnerMinEnd.IsZero() && time.Now().After(spinnerMinEnd) && !m.hosts.reloading {
				spinnerActive = false
				return m, nil
			}
			return m, tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} })
		}
		return m, nil
	}

	// Auto-dismiss toasts.
	if tdm, ok := msg.(toastDismissMsg); ok {
		if tdm.token == m.toastToken {
			m.clearToasts()
		}
		return m, nil
	}

	prev := m.collectToastKey()
	result, cmd := m.doUpdate(msg)
	cur := m.collectToastKey()

	if cur != "" && cur != prev {
		m.toastToken++
		token := m.toastToken
		level := m.maxToastLevel()
		dismiss := tea.Tick(toastDuration(level), func(time.Time) tea.Msg {
			return toastDismissMsg{token: token}
		})
		return result, tea.Batch(cmd, dismiss)
	}
	return result, cmd
}

func (m *appModel) doUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.applyWindowSize(msg)
	case switchScreenMsg:
		m.screen = msg.to
		return m, nil
	case knownHostsReloadMsg:
		// The reload belongs to the Hosts screen even if the user has left it.
		m.opts.Hosts, m.opts.SkippedLines, m.opts.LoadErrors = msg.res.Hosts, msg.res.SkippedLines, msg.errs
		_, cmd := m.hosts.Update(msg)
		return m, cmd
	case openGroupFormMsg:
		var g config.Group
		if msg.index >= 0 && msg.index < len(m.opts.Inventory.Groups) {
			g = m.opts.Inventory.Groups[msg.index]
		}
		m.form = newGroupFormModel(msg.index, g, m.opts.Config.Defaults)
		m.form.parentCrumb = "Groups"
		m.sizeOnOpen(m.form, groupFormModalSize)
		m.screen = screenGroupForm
		return m, nil
	case openGroupFormPrefillMsg:
		m.form = newGroupFormModel(-1, msg.group, m.opts.Config.Defaults)
		m.form.parentCrumb = "Groups"
		m.sizeOnOpen(m.form, groupFormModalSize)
		m.screen = screenGroupForm
		return m, nil
	case groupFormCancelMsg:
		m.form = nil
		m.screen = screenGroups
		return m, nil
	case groupFormSaveMsg:
		if err := m.saveGroup(msg.index, msg.group); err != nil {
			// Keep form open on error.
			m.form.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.groups.toast = toast{text: "saved", level: toastOK}
		m.form = nil
		m.screen = screenGroups
		return m, nil
	case deleteGroupMsg:
		if err := m.deleteGroup(msg.index); err != nil {
			m.groups.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.groups.toast = toast{text: "deleted", level: toastOK}
		return m, nil
	case openGroupHostsMsg:
		if msg.index < 0 || msg.index >= len(m.opts.Inventory.Groups) {
			m.groups.toast = toast{text: "invalid group", level: toastErr}
			return m, nil
		}
		m.gh = newGroupHostsModel(m.opts, msg.index)
		m.sizeOnOpen(m.gh, nil)
		m.screen = screenGroupHosts
		return m, nil
	case openHostPickerMsg:
		m.picker = newHostPickerModel(m.opts)
		m.picker.parentCrumb = m.breadcrumb()
		m.sizeOnOpen(m.picker, pickerModalSize)
		m.returnTo = msg.returnTo
		m.returnGroupIndex = msg.groupIndex
		m.screen = screenHostPicker
		return m, nil
	case openGroupPickerMsg:
		m.gpHosts = append([]string(nil), msg.hosts...)
		m.gp = newGroupPickerModel(m.opts)
		m.gp.hosts = append([]string(nil), m.gpHosts...)
		m.gp.parentCrumb = m.breadcrumb()
		m.gpReturnTo = screenHosts
		m.sizeOnOpen(m.gp, pickerModalSize)
		m.screen = screenGroupPicker
		return m, nil
	case openDefaultsFormMsg:
		m.defaultsForm = newDefaultsFormModel(m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.defaultsReturnTo = msg.returnTo
		m.sizeOnOpen(m.defaultsForm, nil)
		m.screen = screenDefaultsForm
		return m, nil
	case defaultsFormCancelMsg:
		m.defaultsForm = nil
		m.screen = m.defaultsReturnTo
		return m, nil
	case defaultsFormSaveMsg:
		if err := m.saveDefaults(msg.defaults); err != nil {
			m.defaultsForm.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.screen = screenDefaultsForm
		if m.defaultsForm != nil {
			m.defaultsForm.defaults = m.opts.Config.Defaults
			m.defaultsForm.form.resetOriginals()
			m.defaultsForm.toast = toast{text: "saved", level: toastOK}
			m.defaultsToastToken++
			token := m.defaultsToastToken
			return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return defaultsToastExpireMsg{token: token} })
		}
		return m, nil
	case defaultsToastExpireMsg:
		if msg.token != m.defaultsToastToken {
			return m, nil
		}
		// Match on text literal; if the "saved" message ever changes this
		// timer will stop clearing the toast. A sentinel level or token on
		// defaultsFormModel would be cleaner.
		if m.defaultsForm != nil && m.defaultsForm.toast.text == "saved" {
			m.defaultsForm.toast = toast{}
		}
		return m, nil
	case openCustomHostMsg:
		m.customHost = newCustomHostModel(m.opts, msg.groupIndex, msg.returnTo)
		// Build breadcrumb: for screenGroups we need to include the specific
		// group name since m.breadcrumb() only returns "Groups" at that level.
		// For screenGroupHosts, m.breadcrumb() already returns "Groups > name".
		// Other screens don't carry a meaningful groupIndex, so fall through.
		crumb := m.breadcrumb()
		if msg.groupIndex >= 0 && msg.groupIndex < len(m.opts.Inventory.Groups) {
			if m.screen == screenGroups {
				crumb = "Groups > " + m.opts.Inventory.Groups[msg.groupIndex].Name
			}
		}
		m.customHost.parentCrumb = crumb
		m.sizeOnOpen(m.customHost, customHostModalSize)
		m.screen = screenCustomHost
		return m, nil
	case openHostFormMsg:
		idx, hc := findHostConfig(m.opts.Inventory, msg.host)
		m.hostForm = newHostFormModel(idx, hc, m.opts.Config.Defaults)
		m.hostForm.parentCrumb = m.breadcrumb()
		m.hostFormReturnTo = msg.returnTo
		m.sizeOnOpen(m.hostForm, hostFormModalSize)
		m.screen = screenHostForm
		return m, nil
	case openHostFormPrefillMsg:
		m.hostForm = newHostFormModel(-1, msg.host, m.opts.Config.Defaults)
		m.hostForm.parentCrumb = m.breadcrumb()
		m.hostFormReturnTo = msg.returnTo
		m.sizeOnOpen(m.hostForm, hostFormModalSize)
		m.screen = screenHostForm
		return m, nil
	case customHostCancelMsg:
		ret := screenHosts
		if m.customHost != nil {
			ret = m.customHost.returnTo
		}
		m.customHost = nil
		m.screen = ret
		return m, nil
	case customHostPickGroupMsg:
		m.gpHosts = append([]string(nil), msg.hosts...)
		m.gp = newGroupPickerModel(m.opts)
		m.gp.hosts = append([]string(nil), m.gpHosts...)
		m.gp.parentCrumb = m.breadcrumb()
		m.gpReturnTo = msg.returnTo
		m.sizeOnOpen(m.gp, pickerModalSize)
		m.customHost = nil
		m.screen = screenGroupPicker
		return m, nil
	case customHostConnectMsg:
		m.customHost = nil
		m.screen = msg.returnTo
		// Connect through the screen we return to, so typed hosts get the
		// same confirm threshold and off-thread tmux dispatch as listed ones.
		c := m.screenConnect(msg.returnTo)
		var group *config.Group
		if msg.groupIndex >= 0 {
			if msg.groupIndex >= len(m.opts.Inventory.Groups) {
				c.toast = toast{text: "invalid group", level: toastErr}
				return m, nil
			}
			g := m.opts.Inventory.Groups[msg.groupIndex]
			group = &g
		}
		cmd := c.connect(msg.hosts, m.opts, group, false, "")
		if len(c.execCmd) != 0 {
			m.execCmd = c.execCmd
			return m, tea.Quit
		}
		return m, cmd
	case customHostDoneMsg:
		added, err := m.addHostsToGroup(msg.groupIndex, msg.hosts)
		if err != nil {
			if m.customHost != nil {
				m.customHost.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if msg.returnTo == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, msg.groupIndex)
			m.sizeOnOpen(m.gh, nil)
		}
		m.toastOn(msg.returnTo, toast{text: fmt.Sprintf("added %d", added), level: toastOK})
		m.customHost = nil
		m.screen = msg.returnTo
		return m, nil
	case groupPickerCancelMsg:
		m.gp = nil
		m.gpHosts = nil
		m.screen = m.gpReturnTo
		return m, nil
	case groupPickerDoneMsg:
		added, err := m.addHostsToGroup(msg.groupIndex, m.gpHosts)
		if err != nil {
			m.gp.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.toastOn(m.gpReturnTo, toast{text: fmt.Sprintf("added %d", added), level: toastOK})
		m.gp = nil
		m.gpHosts = nil
		m.screen = m.gpReturnTo
		return m, nil
	case hostPickerCancelMsg:
		m.picker = nil
		m.screen = m.returnTo
		return m, nil
	case hostPickerDoneMsg:
		added, err := m.addHostsToGroup(m.returnGroupIndex, msg.hosts)
		if err != nil {
			m.picker.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.picker = nil
		m.screen = m.returnTo
		if m.screen == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, m.returnGroupIndex)
			m.sizeOnOpen(m.gh, nil)
		}
		m.toastOn(m.returnTo, toast{text: fmt.Sprintf("added %d", added), level: toastOK})
		return m, nil
	case removeHostsMsg:
		if err := m.removeHostsFromGroup(msg.groupIndex, msg.hosts); err != nil {
			if m.screen == screenGroupHosts && m.gh != nil {
				m.gh.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.screen == screenGroupHosts && m.gh != nil {
			m.gh.toast = toast{text: fmt.Sprintf("removed %d", len(msg.hosts)), level: toastOK}
		}
		if m.screen == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, msg.groupIndex)
			m.sizeOnOpen(m.gh, nil)
		}
		return m, nil
	case hostFormCancelMsg:
		m.hostForm = nil
		m.screen = m.hostFormReturnTo
		return m, nil
	case hostFormSaveMsg:
		if err := m.saveHostConfig(msg.index, msg.host); err != nil {
			if m.hostForm != nil {
				m.hostForm.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.hosts != nil {
			m.hosts.refreshVisibleBadges(m.opts.Inventory)
			m.hosts.reapplyFilter()
		}
		if m.gh != nil {
			m.gh.refreshVisibleBadges(m.opts.Inventory)
		}
		if m.picker != nil {
			m.picker.refreshVisibleBadges(m.opts.Inventory)
		}
		m.toastOn(m.hostFormReturnTo, toast{text: "saved", level: toastOK})
		m.hostForm = nil
		m.screen = m.hostFormReturnTo
		return m, nil
	case toggleHiddenHostMsg:
		if err := m.saveToggleHidden(msg.host, msg.hide); err != nil {
			if m.hosts != nil {
				m.hosts.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.hosts != nil {
			m.hosts.reapplyFilter()
			m.hosts.toast = toast{text: "saved", level: toastOK}
		}
		return m, nil
	}

	switch m.screen {
	case screenHosts:
		model, cmd := m.hosts.Update(msg)
		if hm, ok := model.(*hostsModel); ok {
			m.hosts = hm
			if len(hm.execCmd) != 0 {
				m.execCmd = hm.execCmd
				return m, tea.Quit
			}
			if hm.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenGroups:
		model, cmd := m.groups.Update(msg)
		if gm, ok := model.(*groupsModel); ok {
			m.groups = gm
			if len(gm.execCmd) != 0 {
				m.execCmd = gm.execCmd
				return m, tea.Quit
			}
			if gm.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenGroupForm:
		model, cmd := m.form.Update(msg)
		if fm, ok := model.(*groupFormModel); ok {
			m.form = fm
		}
		return m, cmd
	case screenGroupHosts:
		model, cmd := m.gh.Update(msg)
		if gh, ok := model.(*groupHostsModel); ok {
			m.gh = gh
			if len(gh.execCmd) != 0 {
				m.execCmd = gh.execCmd
				return m, tea.Quit
			}
			if gh.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenHostPicker:
		model, cmd := m.picker.Update(msg)
		if pm, ok := model.(*hostPickerModel); ok {
			m.picker = pm
		}
		return m, cmd
	case screenGroupPicker:
		model, cmd := m.gp.Update(msg)
		if gm, ok := model.(*groupPickerModel); ok {
			m.gp = gm
		}
		return m, cmd
	case screenDefaultsForm:
		model, cmd := m.defaultsForm.Update(msg)
		if dm, ok := model.(*defaultsFormModel); ok {
			m.defaultsForm = dm
		}
		return m, cmd
	case screenCustomHost:
		model, cmd := m.customHost.Update(msg)
		if cm, ok := model.(*customHostModel); ok {
			m.customHost = cm
		}
		return m, cmd
	case screenHostForm:
		model, cmd := m.hostForm.Update(msg)
		if hm, ok := model.(*hostFormModel); ok {
			m.hostForm = hm
		}
		return m, cmd
	default:
		return m, nil
	}
}

func (m *appModel) View() string {
	var s string
	switch m.screen {
	case screenGroups:
		s = m.groups.View()
	case screenGroupForm:
		s = placeCentered(m.width, m.height, m.form.View())
	case screenGroupHosts:
		s = m.gh.View()
	case screenHostPicker:
		s = placeCentered(m.width, m.height, m.picker.View())
	case screenGroupPicker:
		s = placeCentered(m.width, m.height, m.gp.View())
	case screenDefaultsForm:
		s = m.defaultsForm.View()
	case screenCustomHost:
		s = placeCentered(m.width, m.height, m.customHost.View())
	case screenHostForm:
		s = placeCentered(m.width, m.height, m.hostForm.View())
	default:
		s = m.hosts.View()
	}
	// Apply theme background to the entire screen output so every pixel —
	// including lipgloss.Place padding around modal dialogs — uses the
	// colorscheme background. withThemeBG is a no-op when no theme is active.
	return withThemeBG(s)
}

func (m *appModel) saveGroup(index int, g config.Group) error {
	g.Name = strings.TrimSpace(g.Name)
	groups, err := upsert(m.opts.Inventory.Groups, index, g, func(g config.Group) string { return g.Name }, "group name")
	if err != nil {
		return err
	}
	newInv := m.opts.Inventory
	newInv.Groups = groups
	return m.commitInventory(newInv)
}

// upsert returns a copy of items with item appended (index < 0) or put at
// index. key names an item; a blank or already used name is refused. kind
// names the key in errors.
func upsert[T any](items []T, index int, item T, key func(T) string, kind string) ([]T, error) {
	k := strings.TrimSpace(key(item))
	if k == "" {
		return nil, fmt.Errorf("%s required", kind)
	}
	for i := range items {
		if i != index && strings.TrimSpace(key(items[i])) == k {
			return nil, fmt.Errorf("%s already exists", kind)
		}
	}
	if index >= len(items) {
		return nil, fmt.Errorf("invalid %s index", kind)
	}
	out := append([]T(nil), items...)
	if index < 0 {
		return append(out, item), nil
	}
	out[index] = item
	return out, nil
}

// sizeOnOpen sends a freshly opened screen its first size. size maps the window
// to the screen's modal size; nil means the full window.
func (m *appModel) sizeOnOpen(c tea.Model, size func(w, h int) (int, int)) {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	w, h := m.width, m.height
	if size != nil {
		w, h = size(w, h)
	}
	_, _ = c.Update(tea.WindowSizeMsg{Width: w, Height: h})
}

// toastOn shows t on the list screen s.
func (m *appModel) toastOn(s screen, t toast) { m.screenConnect(s).toast = t }

// screenConnect returns the connect state of the list screen s.
func (m *appModel) screenConnect(s screen) *connectActions {
	switch {
	case s == screenGroups:
		return &m.groups.connectActions
	case s == screenGroupHosts && m.gh != nil:
		return &m.gh.connectActions
	}
	return &m.hosts.connectActions
}

// commitInventory saves newInv and, only if that succeeds, gives it to every
// screen. Callers build newInv on copies of the slices they change, so a
// failed save leaves the current inventory untouched.
func (m *appModel) commitInventory(newInv config.Inventory) error {
	if _, err := config.SaveInventory(m.opts.InventoryPath, &newInv); err != nil {
		return err
	}

	// newAppModel always creates the hosts and groups screens.
	m.opts.Inventory = newInv
	m.hosts.opts.Inventory = newInv
	if !m.opts.Config.Defaults.LoadKnownHosts {
		// Without known_hosts, the host list is the hosts the inventory names.
		m.opts.Hosts = config.ConfigHosts(newInv)
		m.hosts.setHosts(m.opts.Hosts)
	}
	m.groups.Refresh(newInv)
	if m.gh != nil {
		m.gh.opts.Inventory = newInv
	}
	if m.picker != nil {
		m.picker.opts.Inventory = newInv
	}
	if m.gp != nil {
		m.gp.opts.Inventory = newInv
	}
	if m.customHost != nil {
		m.customHost.opts.Inventory = newInv
	}
	return nil
}

func (m *appModel) saveDefaults(d config.Defaults) error {
	oldLoadKnownHosts := m.opts.Config.Defaults.LoadKnownHosts
	oldKnownPaths := append([]string(nil), m.opts.KnownHosts...)

	newCfg := m.opts.Config
	newCfg.Defaults = d
	if _, err := config.Save(m.opts.ConfigPath, &newCfg); err != nil {
		return err
	}

	m.opts.Config = newCfg
	ApplyColorScheme(newCfg.Defaults.Colorscheme, newCfg.Defaults.AccentColor)
	m.refreshAccentStyles()

	// Update hosts source if load_known_hosts toggled.
	if oldLoadKnownHosts != newCfg.Defaults.LoadKnownHosts {
		if newCfg.Defaults.LoadKnownHosts {
			if len(oldKnownPaths) != 0 {
				m.opts.KnownHosts = oldKnownPaths
			}
			if len(m.opts.KnownHosts) == 0 {
				m.opts.KnownHosts = hosts.DefaultKnownHostsPaths()
			}
			res, errs := hosts.LoadKnownHosts(m.opts.KnownHosts)
			m.opts.Hosts = res.Hosts
			m.opts.SkippedLines = res.SkippedLines
			m.opts.LoadErrors = errs
		} else {
			m.opts.KnownHosts = nil
			m.opts.Hosts = config.ConfigHosts(m.opts.Inventory)
			m.opts.SkippedLines = 0
			m.opts.LoadErrors = nil
		}

		if m.hosts != nil {
			m.hosts.keymap.Reload.SetEnabled(newCfg.Defaults.LoadKnownHosts)
			m.hosts.opts = m.opts
			_, _ = m.hosts.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: m.opts.Hosts, SkippedLines: m.opts.SkippedLines}, errs: m.opts.LoadErrors})
		}
		if m.picker != nil {
			// Recreate to refresh list source.
			m.picker = newHostPickerModel(m.opts)
			m.sizeOnOpen(m.picker, pickerModalSize)
		}
	}
	if m.hosts != nil {
		m.hosts.opts = m.opts
		m.hosts.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.groups != nil {
		m.groups.opts = m.opts
		m.groups.Refresh(m.opts.Inventory)
	}
	if m.gh != nil {
		m.gh.opts = m.opts
		m.gh.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.picker != nil {
		m.picker.opts = m.opts
		m.picker.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.gp != nil {
		m.gp.opts = m.opts
	}
	return nil
}

func (m *appModel) refreshAccentStyles() {
	if m.hosts != nil {
		setSearchBarFocused(&m.hosts.search, m.hosts.focus == focusSearch)
		if m.hosts.cmdPrompt {
			setSearchFocused(&m.hosts.cmdInput, true)
		}
	}
	if m.groups != nil {
		setSearchBarFocused(&m.groups.search, m.groups.focus == focusSearch)
		if m.groups.cmdPrompt {
			setSearchFocused(&m.groups.cmdInput, true)
		}
	}
	if m.gh != nil {
		setSearchBarFocused(&m.gh.search, m.gh.focus == focusSearch)
		if m.gh.cmdPrompt {
			setSearchFocused(&m.gh.cmdInput, true)
		}
	}
	if m.picker != nil {
		setSearchBarFocused(&m.picker.search, m.picker.focus == focusSearch)
	}
	if m.gp != nil {
		setSearchBarFocused(&m.gp.search, m.gp.focus == focusSearch)
	}
	if m.customHost != nil {
		setSearchFocused(&m.customHost.input, true)
	}
	if m.defaultsForm != nil {
		m.defaultsForm.refreshAccentStyles()
	}
	if m.form != nil {
		m.form.refreshAccentStyles()
	}
	if m.hostForm != nil {
		m.hostForm.refreshAccentStyles()
	}
}

func (m *appModel) saveHostConfig(index int, h config.Host) error {
	h.Host = strings.TrimSpace(h.Host)
	hs, err := upsert(m.opts.Inventory.Hosts, index, h, func(h config.Host) string { return h.Host }, "host")
	if err != nil {
		return err
	}
	newInv := m.opts.Inventory
	newInv.Hosts = hs
	return m.commitInventory(newInv)
}

func (m *appModel) deleteGroup(index int) error {
	if index < 0 || index >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	newInv.Groups = append(newInv.Groups[:index], newInv.Groups[index+1:]...)
	return m.commitInventory(newInv)
}

// addHostsToGroup adds the hosts the group lacks and returns how many it added.
func (m *appModel) addHostsToGroup(groupIndex int, hostsToAdd []string) (int, error) {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return 0, fmt.Errorf("invalid group index")
	}
	if len(hostsToAdd) == 0 {
		return 0, nil
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	g := newInv.Groups[groupIndex]

	set := make(map[string]bool, len(g.Hosts))
	for _, h := range g.Hosts {
		set[h] = true
	}
	added := 0
	for _, h := range hostsToAdd {
		if h == "" {
			continue
		}
		if set[h] {
			continue
		}
		g.Hosts = append(g.Hosts, h)
		set[h] = true
		added++
	}

	newInv.Groups[groupIndex] = g
	if err := m.commitInventory(newInv); err != nil {
		return 0, err
	}
	return added, nil
}

func (m *appModel) removeHostsFromGroup(groupIndex int, hostsToRemove []string) error {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	if len(hostsToRemove) == 0 {
		return nil
	}

	removeSet := make(map[string]bool, len(hostsToRemove))
	for _, h := range hostsToRemove {
		removeSet[h] = true
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	g := newInv.Groups[groupIndex]
	kept := make([]string, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		if removeSet[h] {
			continue
		}
		kept = append(kept, h)
	}
	g.Hosts = kept
	newInv.Groups[groupIndex] = g
	return m.commitInventory(newInv)
}

func (m *appModel) saveToggleHidden(host string, hide bool) error {
	newInv := m.opts.Inventory
	h := strings.TrimSpace(host)

	// Work on independent copies of mutable slices.
	newInv.HiddenHosts = append([]string(nil), newInv.HiddenHosts...)
	newInv.Hosts = append([]config.Host(nil), newInv.Hosts...)

	idx, hc := findHostConfig(newInv, host)

	if hide {
		if idx >= 0 {
			// Host has existing [[hosts]] entry — set Hidden there.
			hc.Hidden = true
			newInv.Hosts[idx] = hc
		} else {
			// No per-host config — use compact list.
			present := false
			for _, hh := range newInv.HiddenHosts {
				if strings.TrimSpace(hh) == h {
					present = true
					break
				}
			}
			if !present {
				newInv.HiddenHosts = append(newInv.HiddenHosts, h)
			}
		}
	} else {
		// Remove from compact list (safe even if not present).
		out := newInv.HiddenHosts[:0]
		for _, hh := range newInv.HiddenHosts {
			if strings.TrimSpace(hh) != h {
				out = append(out, hh)
			}
		}
		newInv.HiddenHosts = out
		// Also clear Hidden flag in [[hosts]] entry if set (handles old-format configs).
		if idx >= 0 && hc.Hidden {
			hc.Hidden = false
			newInv.Hosts[idx] = hc
		}
	}
	return m.commitInventory(newInv)
}

func (m *appModel) IsQuitting() bool { return m.quitting }
func (m *appModel) ExecCmd() []string {
	return m.execCmd
}
