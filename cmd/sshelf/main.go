package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/hosts"
	"github.com/al-bashkir/sshelf/internal/pane"
	"github.com/al-bashkir/sshelf/internal/ui"
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// cli is the process boundary: output streams, and the two steps that take
// over the terminal (tui) or replace the process (exec). Tests fake them.
type cli struct {
	stdout, stderr io.Writer
	tui            func(ui.Options) error
	exec           func(argv []string) error
}

func main() {
	os.Exit(run(os.Args[1:], cli{stdout: os.Stdout, stderr: os.Stderr, tui: ui.Run, exec: execReplace}))
}

// run executes one sshelf command line and returns the process exit code.
func run(args []string, c cli) int {
	// The pane wrapper needs no flags or config; tmux runs it inside each pane.
	if len(args) > 0 && args[0] == pane.Subcommand {
		return pane.Main(args[1:])
	}
	f, err := parseFlags(args, c.stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return 2 // the flag set already printed the error and usage
	}
	if err := c.dispatch(f); err != nil {
		_, _ = fmt.Fprintln(c.stderr, err)
		return 1
	}
	return 0
}

// globalFlags are the flags that come before a subcommand.
type globalFlags struct {
	configPath, hostsPath string
	knownHosts            multiFlag
	noTmux, popup         bool
	args                  []string // the subcommand and its arguments
}

func parseFlags(args []string, stderr io.Writer) (globalFlags, error) {
	var f globalFlags
	fs := flag.NewFlagSet("sshelf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&f.configPath, "config", "", "path to config.toml (default: XDG config)")
	fs.StringVar(&f.hostsPath, "hosts", "", "path to hosts.toml (default: next to config.toml)")
	fs.Var(&f.knownHosts, "known-hosts", "known_hosts path (repeatable)")
	fs.BoolVar(&f.noTmux, "no-tmux", false, "disable tmux integration")
	fs.BoolVar(&f.popup, "popup", false, "quit after connecting (for tmux popup use)")
	fs.Usage = func() { usage(fs) }
	err := fs.Parse(args)
	f.args = fs.Args()
	return f, err
}

// env is the loaded configuration a command runs against.
type env struct {
	cfg        config.Config
	cfgPath    string
	inv        config.Inventory
	invPath    string
	knownPaths []string
	known      hosts.LoadResult
	loadErrs   []hosts.PathError
}

func load(f globalFlags, stderr io.Writer) (env, error) {
	configPath := f.configPath
	if configPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return env{}, err
		}
		configPath = p
	}

	// Derive hosts.toml path from config path if not explicitly set.
	hostsPath := f.hostsPath
	if hostsPath == "" {
		hostsPath = config.HostsPathFromConfigPath(configPath)
	}

	// Migrate old single-file config if needed. Load after it, so a later
	// save does not see the migration's rewrite as a change on disk.
	if err := config.Migrate(configPath, hostsPath); err != nil {
		_, _ = fmt.Fprintf(stderr, "warning: config migration failed: %v\n", err)
	}

	var e env
	var err error
	if e.cfg, e.cfgPath, err = config.Load(configPath); err != nil {
		return env{}, err
	}
	if e.inv, e.invPath, err = config.LoadInventory(hostsPath); err != nil {
		return env{}, err
	}
	if e.cfg.Defaults.LoadKnownHosts {
		e.knownPaths = f.knownHosts
		e.known, e.loadErrs = hosts.LoadKnownHosts(e.knownPaths)
	} else {
		e.known.Hosts = config.ConfigHosts(e.inv)
	}
	return e, nil
}

func (c cli) dispatch(f globalFlags) error {
	e, err := load(f, c.stderr)
	if err != nil {
		return err
	}
	if len(f.args) == 0 {
		return c.runTUI(ui.Options{
			ConfigPath:    e.cfgPath,
			Config:        e.cfg,
			InventoryPath: e.invPath,
			Inventory:     e.inv,
			KnownHosts:    e.knownPaths,
			Hosts:         e.known.Hosts,
			SkippedLines:  e.known.SkippedLines,
			LoadErrors:    e.loadErrs,
			Popup:         f.popup,
			NoTmux:        f.noTmux,
		})
	}

	args := f.args[1:]
	switch f.args[0] {
	case "connect", "c":
		return c.connect(args, e.cfg, e.inv, f.noTmux)
	case "list", "l":
		return c.list(args, e.inv, e.known.Hosts)
	case "completion", "comp":
		return c.completion(args)
	case "__complete":
		c.complete(args, e.inv, e.known.Hosts)
		return nil
	}
	return fmt.Errorf("unknown command %q\nUsage: sshelf [flags] [connect|list|completion] ...", f.args[0])
}

func (c cli) runTUI(opts ui.Options) error {
	err := c.tui(opts)
	var req *ui.ExecRequest
	if errors.As(err, &req) {
		return c.exec(req.Cmd)
	}
	if errors.Is(err, ui.ErrQuit) {
		return nil
	}
	return err
}

func usage(fs *flag.FlagSet) {
	_, _ = fmt.Fprint(fs.Output(), `sshelf — terminal UI and CLI for SSH connections

Usage:
  sshelf [flags]                        launch interactive TUI
  sshelf [flags] connect host NAME      connect to a host
  sshelf [flags] connect group NAME     connect to all hosts in a group
  sshelf [flags] list hosts             print known hosts
  sshelf [flags] list groups            print configured groups
  sshelf completion bash|zsh            print shell completion script

Subcommand aliases:  connect=c  list=l  host=h  group=g  hosts=h  groups=g

Flags:
`)
	fs.PrintDefaults()
}

func execReplace(cmd []string) error {
	if len(cmd) == 0 {
		return errors.New("empty exec command")
	}
	path, err := exec.LookPath(cmd[0])
	if err != nil {
		return err
	}
	// #nosec G204 -- exec replaces current process with argv (no shell); cmd is constructed by the app.
	return syscall.Exec(path, cmd, os.Environ())
}
