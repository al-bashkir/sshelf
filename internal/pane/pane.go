// Package pane wraps ssh inside a tmux pane. When the connection is lost
// (ssh exits 255) it keeps the pane open and offers to reconnect; any other
// exit means the user ended the session, so the pane closes at once.
package pane

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/x/term"

	tmx "github.com/al-bashkir/sshelf/internal/tmux"
)

// Subcommand is the hidden sshelf subcommand that runs the wrapper.
const Subcommand = "__pane"

// lostCode is the exit code ssh uses for its own errors, including a lost connection.
const lostCode = 255

const prompt = "\r\n\033[41m\033[30m Connection to %s lost: [r] reconnect  [Enter] close \033[0m"

// Action is what the lost-connection prompt does with one key press.
type Action int

const (
	Ignore Action = iota
	Reconnect
	Close
)

// Decide maps a raw key byte to an Action.
func Decide(key byte) Action {
	switch key {
	case 'r', 'R':
		return Reconnect
	case '\r', '\n', 'q', 0x03, 0x04: // Enter, q, Ctrl-C, Ctrl-D
		return Close
	}
	return Ignore
}

// Argv returns the command tmux runs in a pane: exe wraps sshCmd for host.
func Argv(exe, host string, sshCmd []string) []string {
	return append([]string{exe, Subcommand, host, "--"}, sshCmd...)
}

func parseArgs(args []string) (host string, sshCmd []string, err error) {
	if len(args) < 3 || args[1] != "--" {
		return "", nil, errors.New("usage: sshelf __pane HOST -- SSH_ARGV...")
	}
	return args[0], args[2:], nil
}

// Main runs the wrapper with the arguments after Subcommand and returns the exit code.
func Main(args []string) int {
	host, sshCmd, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	// Catch (not ignore) these: ignored signals stay ignored in ssh after exec,
	// caught ones reset to default, so Ctrl-C still reaches ssh.
	signal.Notify(make(chan os.Signal, 1), os.Interrupt, syscall.SIGQUIT)

	paneID := os.Getenv("TMUX_PANE")
	return session{
		host: host,
		ssh:  func() int { return runSSH(sshCmd) },
		key:  readKey,
		isolate: func(on bool) {
			if paneID != "" {
				_ = tmx.IsolatePaneSync(paneID, on)
			}
		},
		out: os.Stdout,
	}.run()
}

type session struct {
	host    string
	ssh     func() int           // runs ssh once, returns its exit code
	key     func() (byte, error) // reads one key press
	isolate func(on bool)        // takes the pane out of (or back into) synchronize-panes
	out     io.Writer
}

func (s session) run() int {
	for {
		code := s.ssh()
		if code != lostCode {
			return code
		}
		s.isolate(true)
		fmt.Fprintf(s.out, prompt, s.host)
		if !s.awaitReconnect() {
			return code
		}
		s.isolate(false)
		fmt.Fprint(s.out, "\r\n")
	}
}

func (s session) awaitReconnect() bool {
	for {
		k, err := s.key()
		if err != nil {
			return false
		}
		switch Decide(k) {
		case Reconnect:
			return true
		case Close:
			return false
		}
	}
}

func runSSH(sshCmd []string) int {
	// #nosec G204 -- argv built by sshelf itself (no shell).
	cmd := exec.Command(sshCmd[0], sshCmd[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr):
		return exitErr.ExitCode()
	}
	fmt.Fprintln(os.Stderr, err)
	return 127
}

// readKey reads one key press without waiting for Enter.
func readKey() (byte, error) {
	fd := os.Stdin.Fd()
	if st, err := term.MakeRaw(fd); err == nil {
		defer func() { _ = term.Restore(fd, st) }()
	}
	var b [1]byte
	if _, err := os.Stdin.Read(b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}
