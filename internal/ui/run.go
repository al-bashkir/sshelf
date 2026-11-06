package ui

import (
	"errors"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrQuit = errors.New("quit")

type ExecRequest struct {
	Cmd []string
}

func (e *ExecRequest) Error() string { return "exec requested" }

type Options struct {
	ConfigPath    string
	Config        config.Config
	InventoryPath string
	Inventory     config.Inventory
	KnownHosts    []string
	Hosts         []string
	SkippedLines  int
	LoadErrors    []hosts.PathError
	Popup         bool // quit after any tmux connect (for tmux popup use)
	NoTmux        bool // --no-tmux: connect in the current pane; never saved to config
}

type exitState interface {
	IsQuitting() bool
	ExecCmd() []string
}

// Run shows the TUI on the real terminal. Its result comes from exitResult.
func Run(opts Options) error {
	opts = withDefaultKnownHosts(opts)
	return exitResult(tea.NewProgram(newAppModel(opts), tea.WithAltScreen()).Run())
}

// withDefaultKnownHosts sets the default known_hosts path when the config
// loads known_hosts and no path was given.
func withDefaultKnownHosts(opts Options) Options {
	if len(opts.KnownHosts) == 0 && opts.Config.Defaults.LoadKnownHosts {
		opts.KnownHosts = hosts.DefaultKnownHostsPaths()
	}
	return opts
}

// exitResult turns how the program ended into Run's error: the program's own
// error, an ExecRequest for the command to exec, ErrQuit, or nil.
func exitResult(model tea.Model, err error) error {
	if err != nil {
		return err
	}
	if st, ok := model.(exitState); ok {
		if cmd := st.ExecCmd(); len(cmd) != 0 {
			return &ExecRequest{Cmd: cmd}
		}
		if st.IsQuitting() {
			return ErrQuit
		}
	}
	return nil
}
