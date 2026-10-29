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

// OpenOneWindow creates a new tmux window and splits it into panes, one per SSH command.
func OpenOneWindow(sshCmds [][]string, opts OneWindowOpts) error {
	if len(sshCmds) == 0 {
		return fmt.Errorf("no hosts selected")
	}

	name := strings.TrimSpace(opts.WindowName)
	if name == "" {
		name = "ssh"
	}

	layout := strings.TrimSpace(opts.Layout)
	if layout == "" {
		layout = "even-horizontal"
	}

	splitFlag := strings.TrimSpace(opts.SplitFlag)
	if splitFlag != "-h" && splitFlag != "-v" {
		splitFlag = "-h"
	}

	borderStatus := strings.TrimSpace(opts.PaneBorderStatus)
	if borderStatus == "" {
		borderStatus = "bottom"
	}
	borderFormat := strings.TrimSpace(opts.PaneBorderFormat)
	if borderFormat == "" {
		borderFormat = config.DefaultPaneBorderFormat
	}

	// Create window and capture both window_id and pane_id.
	args := []string{"new-window", "-P", "-F", "#{window_id} #{pane_id}", "-n", name, "--"}
	args = append(args, sshCmds[0]...)
	// #nosec G204 -- running tmux with argv (no shell); args are constructed by the app.
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux error: %s", tmuxErrMsg(out, err))
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return fmt.Errorf("tmux error: missing window/pane id")
	}
	winID := fields[0]
	firstPaneID := fields[1]

	// Best-effort window settings (do not fail if unsupported).
	_ = tmuxRun("set-window-option", "-t", winID, "automatic-rename", "off")
	_ = tmuxRun("set-window-option", "-t", winID, "allow-rename", "off")
	if strings.EqualFold(borderStatus, "off") {
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-status", "off")
	} else {
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-status", borderStatus)
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-format", borderFormat)
	}
	if opts.SyncPanes && len(sshCmds) > 1 {
		_ = tmuxRun("set-window-option", "-t", winID, "synchronize-panes", "on")
	}

	if title := tmuxPaneTitle(opts.PaneTitles, 0); title != "" {
		_ = tmuxRun("select-pane", "-t", firstPaneID, "-T", title)
	}

	for i := 1; i < len(sshCmds); i++ {
		splitArgs := []string{"split-window", "-t", winID, splitFlag, "-P", "-F", "#{pane_id}", "--"}
		splitArgs = append(splitArgs, sshCmds[i]...)
		// #nosec G204 -- running tmux with argv (no shell); args are constructed by the app.
		out, err := exec.Command("tmux", splitArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("tmux error: %s", tmuxErrMsg(out, err))
		}

		paneFields := strings.Fields(strings.TrimSpace(string(out)))
		paneID := ""
		if len(paneFields) > 0 {
			paneID = paneFields[0]
		}
		if paneID != "" {
			if title := tmuxPaneTitle(opts.PaneTitles, i); title != "" {
				_ = tmuxRun("select-pane", "-t", paneID, "-T", title)
			}
		}
	}

	// Best-effort layout.
	_ = tmuxRun("select-layout", "-t", winID, layout)
	return nil
}

func tmuxPaneTitle(titles []string, idx int) string {
	if idx < 0 || idx >= len(titles) {
		return ""
	}
	return strings.TrimSpace(titles[idx])
}

func tmuxRun(args ...string) error {
	// #nosec G204 -- running tmux with argv (no shell); args are internal.
	_, err := exec.Command("tmux", args...).CombinedOutput()
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
