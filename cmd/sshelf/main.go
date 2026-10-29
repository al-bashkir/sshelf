package main

import (
	"errors"
	"flag"
	"fmt"
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

func main() {
	// The pane wrapper needs no flags or config; tmux runs it inside each pane.
	if len(os.Args) > 1 && os.Args[1] == pane.Subcommand {
		os.Exit(pane.Main(os.Args[2:]))
	}

	var configPath string
	var hostsPath string
	var knownHosts multiFlag
	var noTmux bool
	var popup bool

	flag.StringVar(&configPath, "config", "", "path to config.toml (default: XDG config)")
	flag.StringVar(&hostsPath, "hosts", "", "path to hosts.toml (default: next to config.toml)")
	flag.Var(&knownHosts, "known-hosts", "known_hosts path (repeatable)")
	flag.BoolVar(&noTmux, "no-tmux", false, "disable tmux integration")
	flag.BoolVar(&popup, "popup", false, "quit after connecting (for tmux popup use)")
	flag.Usage = usage
	flag.Parse()

	if configPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			fatal(err)
		}
		configPath = p
	}

	// Derive hosts.toml path from config path if not explicitly set.
	if hostsPath == "" {
		hostsPath = config.HostsPathFromConfigPath(configPath)
	}

	// Migrate old single-file config if needed. Load after it, so a later
	// save does not see the migration's rewrite as a change on disk.
	if err := config.Migrate(configPath, hostsPath); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: config migration failed: %v\n", err)
	}

	cfg, cfgPathUsed, err := config.Load(configPath)
	if err != nil {
		fatal(err)
	}

	inv, invPathUsed, err := config.LoadInventory(hostsPath)
	if err != nil {
		fatal(err)
	}

	knownPaths := []string(knownHosts)
	res := hosts.LoadResult{}
	var loadErrs []hosts.PathError
	if cfg.Defaults.LoadKnownHosts {
		res, loadErrs = hosts.LoadKnownHosts(knownPaths)
	} else {
		knownPaths = nil
		res.Hosts = config.ConfigHosts(inv)
	}

	args := flag.Args()
	if len(args) == 0 {
		runTUI(ui.Options{
			ConfigPath:    cfgPathUsed,
			Config:        cfg,
			InventoryPath: invPathUsed,
			Inventory:     inv,
			KnownHosts:    knownPaths,
			Hosts:         res.Hosts,
			SkippedLines:  res.SkippedLines,
			LoadErrors:    loadErrs,
			Popup:         popup,
			NoTmux:        noTmux,
		})
		return
	}

	switch args[0] {
	case "connect", "c":
		runConnect(args[1:], cfg, inv, noTmux)
	case "list", "l":
		runList(args[1:], inv, res.Hosts)
	case "completion", "comp":
		runCompletion(args[1:])
	case "__complete":
		runInternalComplete(args[1:], inv, res.Hosts)
	default:
		fatal(fmt.Errorf("unknown command %q\nUsage: sshelf [flags] [connect|list|completion] ...", args[0]))
	}
}

func runTUI(opts ui.Options) {
	if err := ui.Run(opts); err != nil {
		var req *ui.ExecRequest
		if errors.As(err, &req) {
			if err := execReplace(req.Cmd); err != nil {
				fatal(err)
			}
			return
		}
		if errors.Is(err, ui.ErrQuit) {
			return
		}
		fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `sshelf — terminal UI and CLI for SSH connections

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
	flag.PrintDefaults()
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
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
