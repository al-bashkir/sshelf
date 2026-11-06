package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/connect"
)

func (c cli) connect(args []string, cfg config.Config, inv config.Inventory, noTmux bool) error {
	if len(args) == 0 {
		return fmt.Errorf("connect requires a subcommand: group|g or host|h\nUsage: sshelf connect group|host NAME")
	}
	switch args[0] {
	case "group", "g":
		if len(args) < 2 {
			return fmt.Errorf("connect group requires a name\nUsage: sshelf connect group NAME")
		}
		return c.connectGroup(args[1], cfg, inv, noTmux)
	case "host", "h":
		if len(args) < 2 {
			return fmt.Errorf("connect host requires a name\nUsage: sshelf connect host NAME")
		}
		return c.open([]string{args[1]}, cfg, inv, nil, noTmux)
	}
	return fmt.Errorf("unknown connect subcommand %q: use group|g or host|h", args[0])
}

func (c cli) connectGroup(name string, cfg config.Config, inv config.Inventory, noTmux bool) error {
	i := slices.IndexFunc(inv.Groups, func(g config.Group) bool { return strings.EqualFold(config.GroupPath(g), name) })
	if i < 0 {
		return fmt.Errorf("group %q not found", name)
	}
	group := config.ResolveGroup(inv.Groups, inv.Groups[i])
	if len(group.Hosts) == 0 {
		return fmt.Errorf("group %q has no hosts", name)
	}
	return c.open(group.Hosts, cfg, inv, &group, noTmux)
}

// open dispatches hosts the same way the TUI does, then either replaces this
// process with ssh/tmux or reports what was opened.
func (c cli) open(hosts []string, cfg config.Config, inv config.Inventory, group *config.Group, noTmux bool) error {
	sshCmds := connect.BuildCommands(hosts, cfg.Defaults, inv, group, nil)
	mode, inTmux := connect.Mode(cfg.Defaults, group, noTmux)

	execCmd, msg, err := connect.Open(hosts, sshCmds, cfg.Defaults, group, mode, inTmux)
	if err != nil {
		return err
	}
	if len(execCmd) != 0 {
		return c.exec(execCmd)
	}
	_, _ = fmt.Fprintln(c.stderr, msg)
	return nil
}
