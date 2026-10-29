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

func Run(opts Options) error {
	if len(opts.KnownHosts) == 0 && opts.Config.Defaults.LoadKnownHosts {
		opts.KnownHosts = hosts.DefaultKnownHostsPaths()
	}

	ApplyColorScheme(opts.Config.Defaults.Colorscheme, opts.Config.Defaults.AccentColor)

	m := newAppModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())

	model, err := p.Run()
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
