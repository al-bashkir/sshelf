package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/al-bashkir/sshelf/internal/config"
)

func runList(args []string, inv config.Inventory, knownHosts []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("list requires a subcommand: groups|g or hosts|h\nUsage: sshelf list groups|hosts [--json]"))
	}

	sub := args[0]
	fs := flag.NewFlagSet("list "+sub, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args[1:]); err != nil {
		fatal(err)
	}

	switch sub {
	case "groups", "g":
		listGroups(inv, *jsonOut)
	case "hosts", "h":
		listHosts(knownHosts, *jsonOut)
	default:
		fatal(fmt.Errorf("unknown list subcommand %q: use groups|g or hosts|h", sub))
	}
}

func listGroups(inv config.Inventory, asJSON bool) {
	if asJSON {
		type groupJSON struct {
			Name  string   `json:"name"`
			Hosts []string `json:"hosts"`
		}
		out := make([]groupJSON, 0, len(inv.Groups))
		for _, g := range inv.Groups {
			hosts := g.Hosts
			if hosts == nil {
				hosts = []string{}
			}
			out = append(out, groupJSON{Name: g.Name, Hosts: hosts})
		}
		printJSON(out)
		return
	}
	for _, g := range inv.Groups {
		fmt.Printf("%s (%d hosts)\n", g.Name, len(g.Hosts))
	}
}

func listHosts(hosts []string, asJSON bool) {
	if asJSON {
		out := hosts
		if out == nil {
			out = []string{}
		}
		printJSON(out)
		return
	}
	for _, h := range hosts {
		fmt.Println(h)
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal(err)
	}
}
