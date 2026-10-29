package ui

import (
	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/connect"
	tmx "github.com/al-bashkir/sshelf/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// dispatchResult holds the outcome of dispatching SSH commands via tmux or direct exec.
type dispatchResult struct {
	// execCmd is set when the caller should exec this command after quitting the TUI.
	execCmd []string
	// quit signals that the TUI should exit (execCmd will be run after).
	quit bool
	// toast is set for in-app feedback (errors or success).
	toast toast
}

// dispatchConnect dispatches SSH commands based on the resolved open mode.
// tmux work is moved into a tea.Cmd so it never blocks the update loop;
// everything else resolves immediately into result.
func dispatchConnect(
	hostsToOpen []string,
	sshCmds [][]string,
	defaults config.Defaults,
	group *config.Group,
	mode tmx.OpenMode,
	inTmux bool,
) (result dispatchResult, cmd tea.Cmd) {
	if !connect.NeedsTmux(mode, inTmux) {
		execCmd, _, err := connect.Open(hostsToOpen, sshCmds, defaults, group, mode, inTmux)
		if err != nil {
			return dispatchResult{toast: toast{text: err.Error(), level: toastWarn}}, nil
		}
		return dispatchResult{execCmd: execCmd, quit: true}, nil
	}

	return dispatchResult{}, func() tea.Msg {
		_, msg, err := connect.Open(hostsToOpen, sshCmds, defaults, group, mode, inTmux)
		if err != nil {
			return toastMsg{text: err.Error(), level: toastErr}
		}
		return toastMsg{text: msg, level: toastInfo}
	}
}
