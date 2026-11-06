package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/al-bashkir/sshelf/internal/config"
)

func (c cli) list(args []string, inv config.Inventory, knownHosts []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list requires a subcommand: groups|g or hosts|h\nUsage: sshelf list groups|hosts [--json]")
	}

	sub := args[0]
	fs := flag.NewFlagSet("list "+sub, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	switch sub {
	case "groups", "g":
		return c.listGroups(inv, *jsonOut)
	case "hosts", "h":
		return c.listHosts(knownHosts, *jsonOut)
	}
	return fmt.Errorf("unknown list subcommand %q: use groups|g or hosts|h", sub)
}

func (c cli) listGroups(inv config.Inventory, asJSON bool) error {
	if asJSON {
		type groupJSON struct {
			Name  string   `json:"name"`
			Hosts []string `json:"hosts"`
		}
		out := make([]groupJSON, 0, len(inv.Groups))
		for _, g := range inv.Groups {
			// A non-nil slice prints as [] instead of null.
			out = append(out, groupJSON{Name: config.GroupPath(g), Hosts: append([]string{}, g.Hosts...)})
		}
		return c.printJSON(out)
	}
	for _, g := range inv.Groups {
		_, _ = fmt.Fprintf(c.stdout, "%s (%d hosts)\n", config.GroupPath(g), len(g.Hosts))
	}
	return nil
}

func (c cli) listHosts(hosts []string, asJSON bool) error {
	if asJSON {
		return c.printJSON(append([]string{}, hosts...)) // [] instead of null
	}
	for _, h := range hosts {
		_, _ = fmt.Fprintln(c.stdout, h)
	}
	return nil
}

func (c cli) printJSON(v any) error {
	enc := json.NewEncoder(c.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
