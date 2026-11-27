// Package connect builds SSH commands and dispatches them, either directly or
// through tmux. It is shared by the TUI and the CLI so both behave identically.
package connect

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/pane"
	"github.com/al-bashkir/sshelf/internal/sshcmd"
	tmx "github.com/al-bashkir/sshelf/internal/tmux"
)

// BuildCommands returns one ssh argv per host, applying settings in
// precedence order: defaults → per-host override → group → modifyFn. Through a
// parent group, a host takes the group of its own that config.HostGroup finds.
func BuildCommands(hosts []string, defaults config.Defaults, inv config.Inventory, group *config.Group, modifyFn func(*sshcmd.Settings)) [][]string {
	base := sshcmd.FromDefaults(defaults)
	out := make([][]string, 0, len(hosts))
	for _, h := range hosts {
		s := base
		if hc, ok := sshcmd.FindHostConfig(inv.Hosts, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		if group != nil {
			s = sshcmd.ApplyGroup(s, config.HostGroup(inv.Groups, *group, h))
		}
		if modifyFn != nil {
			modifyFn(&s)
		}
		out = append(out, sshcmd.BuildCommand(h, s))
	}
	return out
}

// Mode resolves the effective open mode; a group may override the defaults,
// and noTmux (the --no-tmux flag) overrides both.
// The second return value reports whether we are inside a tmux session.
func Mode(defaults config.Defaults, group *config.Group, noTmux bool) (tmx.OpenMode, bool) {
	tmuxSetting := defaults.Tmux
	openModeSetting := defaults.OpenMode
	if group != nil {
		if strings.TrimSpace(group.Tmux) != "" {
			tmuxSetting = group.Tmux
		}
		if strings.TrimSpace(group.OpenMode) != "" {
			openModeSetting = group.OpenMode
		}
	}
	if noTmux {
		tmuxSetting = "never"
	}
	inTmux := tmx.InTmux()
	return tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux), inTmux
}

// NeedsTmux reports whether Open will talk to tmux (and therefore may block).
// Callers that must not stall an event loop use it to move Open off-thread.
func NeedsTmux(mode tmx.OpenMode, inTmux bool) bool {
	return mode != tmx.OpenCurrent && inTmux
}

// Open dispatches sshCmds according to mode. Exactly one of the results is
// meaningful: execCmd is an argv the caller must exec in place of the current
// process, msg describes what was opened in tmux. maxPanes > 0 caps the panes
// per window, so the hosts fill as many windows as they need; 0 puts them all
// in one window.
func Open(hosts []string, sshCmds [][]string, defaults config.Defaults, group *config.Group, mode tmx.OpenMode, inTmux bool, maxPanes int) (execCmd []string, msg string, err error) {
	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires tmux (set open_mode to tmux-window or tmux-pane)")
		}
		return sshCmds[0], "", nil
	}

	sshCmds = wrapForReconnect(hosts, sshCmds)
	if !inTmux {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires an active tmux session")
		}
		return tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0], tmx.HasSession), "", nil
	}

	if usesOneWindow(mode, len(sshCmds)) {
		msg, err = openPaneWindows(hosts, sshCmds, defaults, group, maxPanes)
	} else {
		msg, err = openWindows(hosts, sshCmds, group)
	}
	return nil, msg, err
}

// usesOneWindow reports whether n hosts open as panes of one window
// rather than one window each.
func usesOneWindow(mode tmx.OpenMode, n int) bool {
	return mode == tmx.OpenPane || (mode == tmx.OpenWindow && n > 1)
}

// openPaneWindows opens the hosts as panes, maxPanes per window at most
// (all in one window when maxPanes <= 0).
func openPaneWindows(hosts []string, sshCmds [][]string, defaults config.Defaults, group *config.Group, maxPanes int) (string, error) {
	if maxPanes <= 0 {
		maxPanes = len(sshCmds)
	}
	windows := 0
	for start := 0; start < len(sshCmds); start += maxPanes {
		end := min(start+maxPanes, len(sshCmds))
		if err := openOneWindow(hosts[start:end], sshCmds[start:end], defaults, group); err != nil {
			return "", err
		}
		windows++
	}
	if windows == 1 {
		return fmt.Sprintf("opened %d in one window", len(sshCmds)), nil
	}
	return fmt.Sprintf("opened %d in %d windows", len(sshCmds), windows), nil
}

// openOneWindow opens all hosts as panes in a single new tmux window.
func openOneWindow(hosts []string, sshCmds [][]string, defaults config.Defaults, group *config.Group) error {
	ps := tmx.ResolvePaneSettings(defaults, group, len(sshCmds))
	return tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
		WindowName:       tmx.GroupWindowName(hosts, group),
		PaneTitles:       hosts,
		SplitFlag:        ps.SplitFlag,
		Layout:           ps.Layout,
		SyncPanes:        ps.SyncPanes,
		PaneBorderFormat: ps.BorderFormat,
		PaneBorderStatus: ps.BorderStatus,
	})
}

// openWindows opens one tmux window per host.
func openWindows(hosts []string, sshCmds [][]string, group *config.Group) (string, error) {
	for i, sshCmd := range sshCmds {
		tmuxCmd := tmx.NewWindowCmd(tmx.GroupWindowName(hosts[i:i+1], group), sshCmd)
		// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
		if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
			return "", fmt.Errorf("tmux new-window: %w", err)
		}
	}
	return fmt.Sprintf("opened %d", len(sshCmds)), nil
}

// wrapForReconnect runs each ssh under the sshelf pane wrapper, so a lost
// connection keeps the tmux pane open with a reconnect prompt.
func wrapForReconnect(hosts []string, sshCmds [][]string) [][]string {
	exe, err := os.Executable()
	if err != nil {
		return sshCmds // no wrapper: the pane closes on exit, as before
	}
	out := make([][]string, len(sshCmds))
	for i, c := range sshCmds {
		out[i] = pane.Argv(exe, hosts[i], c)
	}
	return out
}
