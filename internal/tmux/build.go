package tmux

import (
	"os/exec"
	"strings"
)

type OpenMode string

const (
	OpenCurrent OpenMode = "current"
	OpenWindow  OpenMode = "tmux-window"
	OpenPane    OpenMode = "tmux-pane"
)

func ResolveOpenMode(tmuxSetting string, openModeSetting string, inTmux bool) OpenMode {
	tmuxSetting = strings.ToLower(strings.TrimSpace(tmuxSetting))
	openModeSetting = strings.ToLower(strings.TrimSpace(openModeSetting))

	if tmuxSetting == "never" {
		return OpenCurrent
	}

	if tmuxSetting == "force" {
		if openModeSetting == string(OpenPane) {
			return OpenPane
		}
		return OpenWindow
	}

	switch openModeSetting {
	case string(OpenCurrent):
		return OpenCurrent
	case string(OpenWindow):
		return OpenWindow
	case string(OpenPane):
		return OpenPane
	case "", "auto":
		if inTmux {
			return OpenWindow
		}
		return OpenCurrent
	default:
		// Unknown value: keep MVP behavior predictable.
		if inTmux {
			return OpenWindow
		}
		return OpenCurrent
	}
}

func NewWindowCmd(name string, sshCmd []string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "ssh"
	}

	cmd := []string{"tmux", "new-window", "-n", name, "--"}
	cmd = append(cmd, sshCmd...)
	return cmd
}

// NewSessionCmd returns the argv that runs sshCmd in session and attaches to it.
// An existing session gets a new window: "new-session -A" would attach to it
// and silently drop sshCmd.
func NewSessionCmd(session string, sshCmd []string, hasSession func(string) bool) []string {
	session = strings.TrimSpace(session)
	if session == "" {
		session = "sshelf"
	}

	if hasSession(session) {
		// "=" makes tmux match the session name exactly, not by prefix.
		cmd := append([]string{"tmux", "new-window", "-t", "=" + session + ":", "--"}, sshCmd...)
		return append(cmd, ";", "attach-session", "-t", "="+session)
	}
	return append([]string{"tmux", "new-session", "-s", session, "--"}, sshCmd...)
}

// HasSession reports whether a tmux session with exactly this name exists.
func HasSession(session string) bool {
	// #nosec G204 -- tmux argv (no shell); session comes from the user's config.
	return exec.Command("tmux", "has-session", "-t", "="+session).Run() == nil
}
