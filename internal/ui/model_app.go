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
	groupIndex int // -1 means no group context; Ctrl+g will pick a group
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
	gpReturnTo         screen
	returnTo           screen
	returnGroupIndex   int
	defaultsReturnTo   screen
	hostFormReturnTo   screen
	defaultsToastToken int
	toastToken         int
	theme              themeChoice // the colorscheme and accent on screen

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
	m.showTheme(savedTheme(opts.Config.Defaults))
	return m
}

func (m *appModel) Init() tea.Cmd {
	return m.hosts.Init()
}

// screenModel is the model behind one screen and the size it takes from the
// window. A screen with a size is a modal, drawn centered in the window; one
// without fills the window as a tab.
type screenModel struct {
	id    screen
	model tea.Model                 // nil while the screen is not open
	size  func(w, h int) (int, int) // nil: the whole window
}

// screenModels lists every screen in screen order.
func (m *appModel) screenModels() []screenModel {
	return []screenModel{
		{screenHosts, orNil(m.hosts), nil},
		{screenGroups, orNil(m.groups), nil},
		{screenGroupForm, orNil(m.form), groupFormModalSize},
		{screenGroupHosts, orNil(m.gh), nil},
		{screenHostPicker, orNil(m.picker), pickerModalSize},
		{screenGroupPicker, orNil(m.gp), pickerModalSize},
		{screenDefaultsForm, orNil(m.defaultsForm), nil},
		{screenCustomHost, orNil(m.customHost), customHostModalSize},
		{screenHostForm, orNil(m.hostForm), hostFormModalSize},
	}
}

// screenModel returns the entry for screen s; an unknown s gives no model.
func (m *appModel) screenModel(s screen) screenModel {
	for _, sm := range m.screenModels() {
		if sm.id == s {
			return sm
		}
	}
	return screenModel{id: s}
}

// orNil turns a nil model pointer into a nil tea.Model, so callers can test
// the interface for nil.
func orNil[T any, P interface {
	*T
	tea.Model
}](p P) tea.Model {
	if p == nil {
		return nil
	}
	return p
}

// resize sends sm the size it takes from a w x h window.
func (sm screenModel) resize(w, h int) tea.Cmd {
	if sm.size != nil {
		w, h = sm.size(w, h)
	}
	_, cmd := sm.model.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return cmd
}

func (m *appModel) applyWindowSize(ws tea.WindowSizeMsg) tea.Cmd {
	m.width, m.height = ws.Width, ws.Height
	var cmds []tea.Cmd
	for _, sm := range m.screenModels() {
		if sm.model != nil {
			cmds = append(cmds, sm.resize(ws.Width, ws.Height))
		}
	}
	return tea.Batch(cmds...)
}

// toastSlots returns the hosts, groups and group-hosts toasts, in that
// order. A screen that is not open gives an empty toast.
func (m *appModel) toastSlots() [3]toast {
	var s [3]toast
	if m.hosts != nil {
		s[0] = m.hosts.toast
	}
	if m.groups != nil {
		s[1] = m.groups.toast
	}
	if m.gh != nil {
		s[2] = m.gh.toast
	}
	return s
}

// collectToastKey is "" when no toast shows; it changes when a toast does.
func (m *appModel) collectToastKey() string {
	var b strings.Builder
	for i, t := range m.toastSlots() {
		if t.empty() {
			continue
		}
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(t.text)
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
	for _, t := range m.toastSlots() {
		if !t.empty() {
			lvl = max(lvl, t.level)
		}
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

// Update handles msg, then starts a timer to clear the toasts when a new
// one shows.
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prev := m.collectToastKey()
	result, cmd := m.doUpdate(msg)
	m.syncTheme()
	if cur := m.collectToastKey(); cur != "" && cur != prev {
		return result, tea.Batch(cmd, m.toastTimer())
	}
	return result, cmd
}

// toastTimer starts the timer that clears the toasts now showing; the
// worst level sets how long they stay.
func (m *appModel) toastTimer() tea.Cmd {
	m.toastToken++
	token := m.toastToken
	return tea.Tick(toastDuration(m.maxToastLevel()), func(time.Time) tea.Msg {
		return toastDismissMsg{token: token}
	})
}

func (m *appModel) doUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.applyWindowSize(msg)
	case appMsg:
		return m, msg.apply(m)
	}
	return m, m.forward(msg)
}

// forward sends msg to the current screen. Every screen's Update changes
// its model in place. A list screen that asks to exec ssh or to quit ends
// the program.
func (m *appModel) forward(msg tea.Msg) tea.Cmd {
	top := m.screenModel(m.screen).model
	if top == nil {
		return nil
	}
	_, cmd := top.Update(msg)
	if st, ok := top.(exitState); ok {
		return m.exitOr(st, cmd)
	}
	return cmd
}

// exitOr ends the program when st asked to exec ssh or to quit; otherwise
// it returns cmd.
func (m *appModel) exitOr(st exitState, cmd tea.Cmd) tea.Cmd {
	if x := st.ExecCmd(); len(x) != 0 {
		m.execCmd = x
		return tea.Quit
	}
	if st.IsQuitting() {
		m.quitting = true
		return tea.Quit
	}
	return cmd
}

func (m *appModel) View() string {
	sm := m.screenModel(m.screen)
	if sm.model == nil {
		sm = m.screenModel(screenHosts)
	}
	s := sm.model.View()
	if sm.size != nil {
		s = placeCentered(m.width, m.height, s)
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

// sizeOnOpen sends the freshly opened screen s its first size.
func (m *appModel) sizeOnOpen(s screen) {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	_ = m.screenModel(s).resize(m.width, m.height)
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
	return nil
}

// saveDefaults saves the settings and gives them to every list screen.
// Settings opens only from a list screen (Hosts, Groups, Group hosts) and
// every modal closes before it leaves, so no picker or form is open here.
func (m *appModel) saveDefaults(d config.Defaults) error {
	cfg := m.opts.Config
	cfg.Defaults = d
	if _, err := config.Save(m.opts.ConfigPath, &cfg); err != nil {
		return err
	}
	toggled := d.LoadKnownHosts != m.opts.Config.Defaults.LoadKnownHosts
	m.opts.Config = cfg
	if toggled {
		m.reloadHostSource()
	}
	m.hosts.opts = m.opts
	m.groups.opts = m.opts
	if m.gh != nil {
		m.gh.opts = m.opts
	}
	return nil
}

// reloadHostSource reads the host list again after load_known_hosts
// changed: the known_hosts files when it is on, the inventory when off.
func (m *appModel) reloadHostSource() {
	on := m.opts.Config.Defaults.LoadKnownHosts
	if on {
		if len(m.opts.KnownHosts) == 0 {
			m.opts.KnownHosts = hosts.DefaultKnownHostsPaths()
		}
		res, errs := hosts.LoadKnownHosts(m.opts.KnownHosts)
		m.opts.Hosts, m.opts.SkippedLines, m.opts.LoadErrors = res.Hosts, res.SkippedLines, errs
	} else {
		m.opts.KnownHosts = nil
		m.opts.Hosts, m.opts.SkippedLines, m.opts.LoadErrors = config.ConfigHosts(m.opts.Inventory), 0, nil
	}
	m.hosts.keymap.Reload.SetEnabled(on)
	m.hosts.opts = m.opts
	_, _ = m.hosts.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: m.opts.Hosts, SkippedLines: m.opts.SkippedLines}, errs: m.opts.LoadErrors})
}

// syncTheme shows the theme Settings previews while Settings is on screen,
// and the saved theme everywhere else.
func (m *appModel) syncTheme() {
	want := savedTheme(m.opts.Config.Defaults)
	if m.screen == screenDefaultsForm && m.defaultsForm != nil {
		want = m.defaultsForm.previewTheme()
	}
	if want != m.theme {
		m.showTheme(want)
	}
}

// showTheme applies t and restyles every open screen.
func (m *appModel) showTheme(t themeChoice) {
	m.theme = t
	ApplyColorScheme(t.scheme, t.accent)
	m.refreshAccentStyles()
}

// refreshAccentStyles restyles every open screen after the accent changed.
func (m *appModel) refreshAccentStyles() {
	for _, sm := range m.screenModels() {
		if s, ok := sm.model.(interface{ refreshAccentStyles() }); ok {
			s.refreshAccentStyles()
		}
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
	if hide {
		return m.commitInventory(hideHost(m.opts.Inventory, host))
	}
	return m.commitInventory(showHost(m.opts.Inventory, host))
}

func (m *appModel) IsQuitting() bool { return m.quitting }
func (m *appModel) ExecCmd() []string {
	return m.execCmd
}
