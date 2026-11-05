package ui

import (
	"fmt"
	"time"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// appMsg is a message the app router handles itself: it opens, closes or
// saves a screen. apply does the work and returns the next command. Every
// other message goes to the current screen (see appModel.forward).
type appMsg interface {
	apply(m *appModel) tea.Cmd
}

// enter makes the freshly opened screen s current and sends it its size.
func (m *appModel) enter(s screen) {
	m.sizeOnOpen(s)
	m.screen = s
}

// group returns group i of the inventory; ok is false when there is none.
func (m *appModel) group(i int) (g config.Group, ok bool) {
	if i < 0 || i >= len(m.opts.Inventory.Groups) {
		return config.Group{}, false
	}
	return m.opts.Inventory.Groups[i], true
}

// reopenGroupHosts rebuilds the group hosts screen on group i, so it shows
// the saved inventory.
func (m *appModel) reopenGroupHosts(i int) {
	m.gh = newGroupHostsModel(m.opts, i)
	m.sizeOnOpen(screenGroupHosts)
}

// returnAfterAdd goes back to s after added hosts went into group i and
// says how many. Group hosts is rebuilt to list them.
func (m *appModel) returnAfterAdd(s screen, i, added int) {
	if s == screenGroupHosts {
		m.reopenGroupHosts(i)
	}
	m.toastOn(s, toast{text: fmt.Sprintf("added %d", added), level: toastOK})
	m.screen = s
}

func (spinnerTickMsg) apply(m *appModel) tea.Cmd {
	if !spinnerActive {
		return nil
	}
	spinnerIndex++
	// Stop once the minimum time has passed and the reload is done.
	if !spinnerMinEnd.IsZero() && time.Now().After(spinnerMinEnd) && !m.hosts.reloading {
		spinnerActive = false
		return nil
	}
	return tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

// toastDismissMsg clears the toasts only for the latest timer; a newer
// toast restarted the timer and keeps its own time.
func (msg toastDismissMsg) apply(m *appModel) tea.Cmd {
	if msg.token == m.toastToken {
		m.clearToasts()
	}
	return nil
}

func (msg switchScreenMsg) apply(m *appModel) tea.Cmd {
	m.screen = msg.to
	return nil
}

func (msg knownHostsReloadMsg) apply(m *appModel) tea.Cmd {
	// The reload belongs to the Hosts screen even if the user has left it.
	m.opts.Hosts, m.opts.SkippedLines, m.opts.LoadErrors = msg.res.Hosts, msg.res.SkippedLines, msg.errs
	_, cmd := m.hosts.Update(msg)
	return cmd
}

// Groups and the group form.

func (m *appModel) openGroupForm(index int, g config.Group) {
	m.form = newGroupFormModel(index, g, m.opts.Config.Defaults)
	m.form.parentCrumb = "Groups"
	m.enter(screenGroupForm)
}

func (msg openGroupFormMsg) apply(m *appModel) tea.Cmd {
	g, _ := m.group(msg.index) // out of range: an empty form
	m.openGroupForm(msg.index, g)
	return nil
}

func (msg openGroupFormPrefillMsg) apply(m *appModel) tea.Cmd {
	m.openGroupForm(-1, msg.group)
	return nil
}

func (groupFormCancelMsg) apply(m *appModel) tea.Cmd {
	m.form = nil
	m.screen = screenGroups
	return nil
}

func (msg groupFormSaveMsg) apply(m *appModel) tea.Cmd {
	if err := m.saveGroup(msg.index, msg.group); err != nil {
		// Keep form open on error.
		m.form.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.groups.toast = toast{text: "saved", level: toastOK}
	m.form = nil
	m.screen = screenGroups
	return nil
}

func (msg deleteGroupMsg) apply(m *appModel) tea.Cmd {
	if err := m.deleteGroup(msg.index); err != nil {
		m.groups.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.groups.toast = toast{text: "deleted", level: toastOK}
	return nil
}

// Group hosts and the host picker that adds to it.

func (msg openGroupHostsMsg) apply(m *appModel) tea.Cmd {
	if _, ok := m.group(msg.index); !ok {
		m.groups.toast = toast{text: "invalid group", level: toastErr}
		return nil
	}
	m.gh = newGroupHostsModel(m.opts, msg.index)
	m.enter(screenGroupHosts)
	return nil
}

func (msg removeHostsMsg) apply(m *appModel) tea.Cmd {
	err := m.removeHostsFromGroup(msg.groupIndex, msg.hosts)
	if m.screen != screenGroupHosts {
		return nil
	}
	if err != nil {
		m.gh.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.reopenGroupHosts(msg.groupIndex)
	m.gh.toast = toast{text: fmt.Sprintf("removed %d", len(msg.hosts)), level: toastOK}
	return nil
}

func (msg openHostPickerMsg) apply(m *appModel) tea.Cmd {
	m.picker = newHostPickerModel(m.opts)
	m.picker.parentCrumb = m.breadcrumb()
	m.returnTo = msg.returnTo
	m.returnGroupIndex = msg.groupIndex
	m.enter(screenHostPicker)
	return nil
}

func (hostPickerCancelMsg) apply(m *appModel) tea.Cmd {
	m.picker = nil
	m.screen = m.returnTo
	return nil
}

func (msg hostPickerDoneMsg) apply(m *appModel) tea.Cmd {
	added, err := m.addHostsToGroup(m.returnGroupIndex, msg.hosts)
	if err != nil {
		m.picker.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.picker = nil
	m.returnAfterAdd(m.returnTo, m.returnGroupIndex, added)
	return nil
}

// The group picker: add hosts to a group the user picks.

// openGroupPicker opens the group picker on hosts; it goes back to returnTo.
func (m *appModel) openGroupPicker(hosts []string, returnTo screen) {
	m.gp = newGroupPickerModel(m.opts)
	m.gp.hosts = append([]string(nil), hosts...)
	m.gp.parentCrumb = m.breadcrumb()
	m.gpReturnTo = returnTo
	m.enter(screenGroupPicker)
}

func (msg openGroupPickerMsg) apply(m *appModel) tea.Cmd {
	m.openGroupPicker(msg.hosts, screenHosts)
	return nil
}

func (groupPickerCancelMsg) apply(m *appModel) tea.Cmd {
	m.gp = nil
	m.screen = m.gpReturnTo
	return nil
}

func (msg groupPickerDoneMsg) apply(m *appModel) tea.Cmd {
	added, err := m.addHostsToGroup(msg.groupIndex, m.gp.hosts)
	if err != nil {
		m.gp.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.toastOn(m.gpReturnTo, toast{text: fmt.Sprintf("added %d", added), level: toastOK})
	m.gp = nil
	m.screen = m.gpReturnTo
	return nil
}

// Settings.

func (msg openDefaultsFormMsg) apply(m *appModel) tea.Cmd {
	m.defaultsForm = newDefaultsFormModel(m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
	m.defaultsReturnTo = msg.returnTo
	m.enter(screenDefaultsForm)
	return nil
}

func (defaultsFormCancelMsg) apply(m *appModel) tea.Cmd {
	m.defaultsForm = nil
	m.screen = m.defaultsReturnTo
	return nil
}

func (msg defaultsFormSaveMsg) apply(m *appModel) tea.Cmd {
	if err := m.saveDefaults(msg.defaults); err != nil {
		m.defaultsForm.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.screen = screenDefaultsForm
	if m.defaultsForm == nil {
		return nil
	}
	m.defaultsForm.defaults = m.opts.Config.Defaults
	m.defaultsForm.form.resetOriginals()
	m.defaultsForm.toast = toast{text: "saved", level: toastOK}
	m.defaultsToastToken++
	token := m.defaultsToastToken
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return defaultsToastExpireMsg{token: token} })
}

func (msg defaultsToastExpireMsg) apply(m *appModel) tea.Cmd {
	// Match on text literal; if the "saved" message ever changes this
	// timer will stop clearing the toast. A sentinel level or token on
	// defaultsFormModel would be cleaner.
	if msg.token == m.defaultsToastToken && m.defaultsForm != nil && m.defaultsForm.toast.text == "saved" {
		m.defaultsForm.toast = toast{}
	}
	return nil
}

// Custom host: hosts the user types.

func (msg openCustomHostMsg) apply(m *appModel) tea.Cmd {
	m.customHost = newCustomHostModel(m.opts, msg.groupIndex, msg.returnTo)
	// On Groups the crumb names the group, since m.breadcrumb() only gives
	// "Groups" there. On Group hosts m.breadcrumb() already names it. Other
	// screens do not carry a meaningful groupIndex.
	m.customHost.parentCrumb = m.breadcrumb()
	if g, ok := m.group(msg.groupIndex); ok && m.screen == screenGroups {
		m.customHost.parentCrumb = "Groups > " + g.Name
	}
	m.enter(screenCustomHost)
	return nil
}

func (customHostCancelMsg) apply(m *appModel) tea.Cmd {
	ret := screenHosts
	if m.customHost != nil {
		ret = m.customHost.returnTo
	}
	m.customHost = nil
	m.screen = ret
	return nil
}

func (msg customHostPickGroupMsg) apply(m *appModel) tea.Cmd {
	m.openGroupPicker(msg.hosts, msg.returnTo)
	m.customHost = nil
	return nil
}

func (msg customHostConnectMsg) apply(m *appModel) tea.Cmd {
	m.customHost = nil
	m.screen = msg.returnTo
	// Connect through the screen we return to, so typed hosts get the
	// same confirm threshold and off-thread tmux dispatch as listed ones.
	c := m.screenConnect(msg.returnTo)
	var group *config.Group
	if msg.groupIndex >= 0 {
		g, ok := m.group(msg.groupIndex)
		if !ok {
			c.toast = toast{text: "invalid group", level: toastErr}
			return nil
		}
		group = &g
	}
	return m.exitOr(c, c.connect(msg.hosts, m.opts, group, false, ""))
}

func (msg customHostDoneMsg) apply(m *appModel) tea.Cmd {
	added, err := m.addHostsToGroup(msg.groupIndex, msg.hosts)
	if err != nil {
		if m.customHost != nil {
			m.customHost.toast = toast{text: err.Error(), level: toastErr}
		}
		return nil
	}
	m.customHost = nil
	m.returnAfterAdd(msg.returnTo, msg.groupIndex, added)
	return nil
}

// Host config and hidden hosts.

func (m *appModel) openHostForm(index int, h config.Host, returnTo screen) {
	m.hostForm = newHostFormModel(index, h, m.opts.Config.Defaults)
	m.hostForm.parentCrumb = m.breadcrumb()
	m.hostFormReturnTo = returnTo
	m.enter(screenHostForm)
}

func (msg openHostFormMsg) apply(m *appModel) tea.Cmd {
	idx, hc := findHostConfig(m.opts.Inventory, msg.host)
	m.openHostForm(idx, hc, msg.returnTo)
	return nil
}

func (msg openHostFormPrefillMsg) apply(m *appModel) tea.Cmd {
	m.openHostForm(-1, msg.host, msg.returnTo)
	return nil
}

func (hostFormCancelMsg) apply(m *appModel) tea.Cmd {
	m.hostForm = nil
	m.screen = m.hostFormReturnTo
	return nil
}

func (msg hostFormSaveMsg) apply(m *appModel) tea.Cmd {
	if err := m.saveHostConfig(msg.index, msg.host); err != nil {
		if m.hostForm != nil {
			m.hostForm.toast = toast{text: err.Error(), level: toastErr}
		}
		return nil
	}
	// The saved config changes the has-config badge on every host list.
	m.hosts.refreshVisibleBadges(m.opts.Inventory)
	m.hosts.reapplyFilter()
	if m.gh != nil {
		m.gh.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.picker != nil {
		m.picker.refreshVisibleBadges(m.opts.Inventory)
	}
	m.toastOn(m.hostFormReturnTo, toast{text: "saved", level: toastOK})
	m.hostForm = nil
	m.screen = m.hostFormReturnTo
	return nil
}

func (msg toggleHiddenHostMsg) apply(m *appModel) tea.Cmd {
	if err := m.saveToggleHidden(msg.host, msg.hide); err != nil {
		m.hosts.toast = toast{text: err.Error(), level: toastErr}
		return nil
	}
	m.hosts.reapplyFilter()
	m.hosts.toast = toast{text: "saved", level: toastOK}
	return nil
}
