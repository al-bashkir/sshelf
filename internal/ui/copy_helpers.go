package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

func suggestCopyName(name string, exists func(string) bool) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "copy"
	}

	c1 := base + "-copy"
	if !exists(c1) {
		return c1
	}
	for i := 2; i < 1000; i++ {
		c := fmt.Sprintf("%s-copy%d", base, i)
		if !exists(c) {
			return c
		}
	}
	return c1
}

func suggestCopyGroupName(inv config.Inventory, from string) string {
	exists := func(n string) bool {
		n = strings.TrimSpace(n)
		for _, g := range inv.Groups {
			if strings.TrimSpace(g.Name) == n {
				return true
			}
		}
		return false
	}
	return suggestCopyName(from, exists)
}

func suggestCopyHostKey(inv config.Inventory, from string) string {
	exists := func(n string) bool {
		n = strings.TrimSpace(n)
		for _, h := range inv.Hosts {
			if strings.TrimSpace(h.Host) == n {
				return true
			}
		}
		return false
	}
	return suggestCopyName(from, exists)
}
