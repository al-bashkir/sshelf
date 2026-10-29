package ui

import (
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/sshcmd"

	tea "github.com/charmbracelet/bubbletea"
)

// openHostConfigCmd opens the per-host config form for host.
func openHostConfigCmd(host string, returnTo screen, t *toast) tea.Cmd {
	if strings.TrimSpace(host) == "" {
		*t = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	return func() tea.Msg { return openHostFormMsg{host: host, returnTo: returnTo} }
}

// copyHostConfigCmd prefills a new host form from host's existing config.
func copyHostConfigCmd(inv config.Inventory, host string, returnTo screen, t *toast) tea.Cmd {
	if strings.TrimSpace(host) == "" {
		*t = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	hc, ok := sshcmd.FindHostConfig(inv.Hosts, host)
	if !ok {
		*t = toast{text: "no host config", level: toastWarn}
		return nil
	}
	hc.Host = suggestCopyHostKey(inv, hc.Host)
	return func() tea.Msg { return openHostFormPrefillMsg{host: hc, returnTo: returnTo} }
}

func findHostConfig(inv config.Inventory, host string) (index int, hc config.Host) {
	if h, ok := sshcmd.FindHostConfig(inv.Hosts, host); ok {
		for i := range inv.Hosts {
			if strings.TrimSpace(inv.Hosts[i].Host) == strings.TrimSpace(h.Host) {
				return i, h
			}
		}
		return -1, h
	}
	return -1, config.Host{Host: strings.TrimSpace(host)}
}

func isHostHidden(inv config.Inventory, host string) bool {
	h := strings.TrimSpace(host)
	for _, hh := range inv.HiddenHosts {
		if strings.TrimSpace(hh) == h {
			return true
		}
	}
	hc, ok := sshcmd.FindHostConfig(inv.Hosts, host)
	return ok && hc.Hidden
}

// hostIndex answers FindHostConfig and isHostHidden in O(1). Built once per
// filter pass so a pass costs O(hosts + inventory), not the product.
type hostIndex struct {
	cfg    map[string]config.Host
	hidden map[string]struct{}
}

func newHostIndex(inv config.Inventory) hostIndex {
	x := hostIndex{
		cfg:    make(map[string]config.Host, len(inv.Hosts)),
		hidden: make(map[string]struct{}, len(inv.HiddenHosts)),
	}
	for _, h := range inv.Hosts {
		k := strings.TrimSpace(h.Host)
		if _, dup := x.cfg[k]; !dup { // first entry wins, as in FindHostConfig
			x.cfg[k] = h
		}
	}
	for _, h := range inv.HiddenHosts {
		x.hidden[strings.TrimSpace(h)] = struct{}{}
	}
	return x
}

// lookup mirrors sshcmd.FindHostConfig, including the "[host]:port" fallback.
func (x hostIndex) lookup(host string) (config.Host, bool) {
	h := strings.TrimSpace(host)
	if h == "" {
		return config.Host{}, false
	}
	if hc, ok := x.cfg[h]; ok {
		return hc, true
	}
	if base, ok := sshcmd.BracketHostBase(h); ok {
		hc, ok := x.cfg[base]
		return hc, ok
	}
	return config.Host{}, false
}

func (x hostIndex) isHidden(host string) bool {
	if _, ok := x.hidden[strings.TrimSpace(host)]; ok {
		return true
	}
	hc, ok := x.lookup(host)
	return ok && hc.Hidden
}
