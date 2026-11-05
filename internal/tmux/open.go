package tmux

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

// OneWindowOpts controls how tmuxOpenOneWindow creates panes in a single tmux window.
type OneWindowOpts struct {
	WindowName string
	PaneTitles []string

	// SplitFlag is passed to tmux split-window: "-h" or "-v".
	SplitFlag string
	// Layout to apply after creating panes (example: "even-horizontal").
	Layout string

	// If true, enable synchronize-panes for the window.
	SyncPanes bool

	// PaneBorderFormat and PaneBorderStatus are tmux window options.
	PaneBorderFormat string
	PaneBorderStatus string // off|top|bottom
}

// tmuxExec runs one tmux command and returns its combined output.
type tmuxExec func(args ...string) ([]byte, error)

func execTmux(args ...string) ([]byte, error) {
	// #nosec G204 -- running tmux with argv (no shell); args are constructed by the app.
	return exec.Command("tmux", args...).CombinedOutput()
}

// OpenOneWindow creates a new tmux window and splits it into panes, one per SSH command.
func OpenOneWindow(sshCmds [][]string, opts OneWindowOpts) error {
	return openOneWindow(execTmux, sshCmds, opts)
}

func openOneWindow(run tmuxExec, sshCmds [][]string, opts OneWindowOpts) error {
	if len(sshCmds) == 0 {
		return fmt.Errorf("no hosts selected")
	}
	o := withDefaults(opts)

	// Create window and capture both window_id and pane_id.
	ids, err := runIDs(run, append([]string{"new-window", "-P", "-F", "#{window_id} #{pane_id}", "-n", o.WindowName, "--"}, sshCmds[0]...))
	if err != nil {
		return err
	}
	if len(ids) < 2 {
		return fmt.Errorf("tmux error: missing window/pane id")
	}
	winID := ids[0]

	// Best-effort window settings (do not fail if unsupported).
	for _, args := range windowOptionArgs(winID, o, len(sshCmds)) {
		_, _ = run(args...)
	}
	titlePane(run, ids[1], o.PaneTitles, 0)

	for i := 1; i < len(sshCmds); i++ {
		ids, err := runIDs(run, append([]string{"split-window", "-t", winID, o.SplitFlag, "-P", "-F", "#{pane_id}", "--"}, sshCmds[i]...))
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			titlePane(run, ids[0], o.PaneTitles, i)
		}
	}

	// Best-effort layout.
	_, _ = run("select-layout", "-t", winID, o.Layout)
	return nil
}

// withDefaults trims opts and replaces empty or unknown values with defaults.
func withDefaults(o OneWindowOpts) OneWindowOpts {
	o.WindowName = orDefault(o.WindowName, "ssh")
	o.Layout = orDefault(o.Layout, "even-horizontal")
	if o.SplitFlag = strings.TrimSpace(o.SplitFlag); o.SplitFlag != "-v" {
		o.SplitFlag = "-h"
	}
	o.PaneBorderStatus = orDefault(o.PaneBorderStatus, "bottom")
	o.PaneBorderFormat = orDefault(o.PaneBorderFormat, config.DefaultPaneBorderFormat)
	return o
}

func orDefault(s, def string) string {
	if s = strings.TrimSpace(s); s != "" {
		return s
	}
	return def
}

// windowOptionArgs returns the set-window-option commands for a new window.
func windowOptionArgs(winID string, o OneWindowOpts, panes int) [][]string {
	set := func(kv ...string) []string {
		return append([]string{"set-window-option", "-t", winID}, kv...)
	}
	args := [][]string{set("automatic-rename", "off"), set("allow-rename", "off")}
	if strings.EqualFold(o.PaneBorderStatus, "off") {
		args = append(args, set("pane-border-status", "off"))
	} else {
		args = append(args, set("pane-border-status", o.PaneBorderStatus), set("pane-border-format", o.PaneBorderFormat))
	}
	if o.SyncPanes && panes > 1 {
		args = append(args, set("synchronize-panes", "on"))
	}
	return args
}

// runIDs runs a tmux command that prints ids and returns them.
func runIDs(run tmuxExec, args []string) ([]string, error) {
	out, err := run(args...)
	if err != nil {
		return nil, fmt.Errorf("tmux error: %s", tmuxErrMsg(out, err))
	}
	return strings.Fields(string(out)), nil
}

func titlePane(run tmuxExec, paneID string, titles []string, idx int) {
	if title := tmuxPaneTitle(titles, idx); title != "" {
		_, _ = run("select-pane", "-t", paneID, "-T", title)
	}
}

func tmuxPaneTitle(titles []string, idx int) string {
	if idx < 0 || idx >= len(titles) {
		return ""
	}
	return strings.TrimSpace(titles[idx])
}

func tmuxRun(args ...string) error {
	_, err := execTmux(args...)
	return err
}

func tmuxErrMsg(out []byte, err error) string {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err.Error()
	}
	return msg
}

// IsolatePaneSync turns synchronize-panes off for one pane only (tmux >= 3.2),
// so keys typed in that pane and in the others do not cross. With
// isolate=false the pane goes back to the window setting.
func IsolatePaneSync(paneID string, isolate bool) error {
	if isolate {
		return tmuxRun("set-option", "-p", "-t", paneID, "synchronize-panes", "off")
	}
	return tmuxRun("set-option", "-p", "-u", "-t", paneID, "synchronize-panes")
}
