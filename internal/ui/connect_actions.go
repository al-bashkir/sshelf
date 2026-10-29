package ui

import (
	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/connect"
	"github.com/al-bashkir/sshelf/internal/sshcmd"
	tmx "github.com/al-bashkir/sshelf/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// connectActions is the connect behaviour shared by every list screen
// (Hosts, Group Hosts, Groups). Screens embed it, so its fields are promoted
// and referenced as m.toast, m.execCmd, m.confirmConnect, ...
type connectActions struct {
	toast    toast
	execCmd  []string
	quitting bool

	confirmConnect      bool
	confirmConnectCount int
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd
}

func connectThreshold(d config.Defaults) int {
	if d.ConnectConfirmThreshold < 0 {
		return 5
	}
	return d.ConnectConfirmThreshold
}

// connect dispatches hosts. oneWindow forces every host into a single tmux
// window of panes; remoteCmd, when set, runs on the remote and keeps the
// session open. Above the confirm threshold the dispatch is deferred until
// the user answers the confirm dialog.
func (c *connectActions) connect(hosts []string, opts Options, group *config.Group, oneWindow bool, remoteCmd string) tea.Cmd {
	if len(hosts) == 0 {
		c.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if oneWindow && !tmx.InTmux() {
		c.toast = toast{text: "requires an active tmux session", level: toastWarn}
		return nil
	}

	defaults := opts.Config.Defaults
	doConnect := func() tea.Cmd {
		mode, inTmux := connect.Mode(defaults, group, opts.NoTmux)
		if oneWindow {
			mode = tmx.OpenPane
		}
		sshCmds := connect.BuildCommands(hosts, defaults, opts.Inventory, group, remoteCmdModifier(remoteCmd))

		res, cmd := dispatchConnect(hosts, sshCmds, defaults, group, mode, inTmux)
		if !res.toast.empty() {
			c.toast = res.toast
		}
		if res.quit {
			c.execCmd = res.execCmd
			return tea.Quit
		}
		return cmd
	}

	if len(hosts) > connectThreshold(defaults) {
		c.confirmConnect = true
		c.confirmConnectCount = len(hosts)
		c.confirmConnectHosts = hosts
		c.pendingConnectFn = doConnect
		return nil
	}
	return doConnect()
}

// connectSameWindow replaces the TUI with a single ssh session in the current pane.
func (c *connectActions) connectSameWindow(hosts []string, opts Options, group *config.Group) tea.Cmd {
	if len(hosts) == 0 {
		c.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if len(hosts) > 1 {
		c.toast = toast{text: "select single host for same-window connect", level: toastWarn}
		return nil
	}
	c.execCmd = connect.BuildCommands(hosts, opts.Config.Defaults, opts.Inventory, group, nil)[0]
	return tea.Quit
}

// remoteCmdModifier returns the settings tweak that runs cmd on the remote
// host and drops into a shell afterwards, or nil when cmd is empty.
func remoteCmdModifier(cmd string) func(*sshcmd.Settings) {
	if keepSessionOpenRemoteCmd(cmd) == "" {
		return nil
	}
	return func(s *sshcmd.Settings) {
		s.ExtraArgs = ensureSSHForceTTY(s.ExtraArgs)
		s.RemoteCommand = keepSessionOpenRemoteCmd(cmd)
	}
}
