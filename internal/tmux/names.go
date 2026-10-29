package tmux

import (
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

// WindowName returns a sanitized tmux window name for the given host address.
func WindowName(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "[")
	host = strings.ReplaceAll(host, "]", "")
	host = strings.ReplaceAll(host, ":", "_")
	if host == "" {
		return "ssh"
	}
	if len(host) > 30 {
		host = host[:30]
	}
	return host
}

// GroupWindowName returns the window name for a set of hosts, preferring the group name when set.
func GroupWindowName(hosts []string, group *config.Group) string {
	if group != nil {
		if n := strings.TrimSpace(group.Name); n != "" {
			return n
		}
	}
	if len(hosts) == 0 {
		return "ssh"
	}
	return WindowName(hosts[0])
}
