package ui

import (
	"fmt"
	"slices"
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
	return suggestCopyName(from, taken(inv.Groups, func(g config.Group) string { return g.Name }))
}

func suggestCopyHostKey(inv config.Inventory, from string) string {
	return suggestCopyName(from, taken(inv.Hosts, func(h config.Host) string { return h.Host }))
}

// taken reports whether a candidate name equals the trimmed key of an item.
// Candidates from suggestCopyName are already trimmed.
func taken[T any](items []T, key func(T) string) func(string) bool {
	return func(name string) bool {
		return slices.ContainsFunc(items, func(it T) bool { return strings.TrimSpace(key(it)) == name })
	}
}
