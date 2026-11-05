package ui

import (
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestSuggestCopyNames(t *testing.T) {
	inv := config.Inventory{
		Groups: []config.Group{{Name: "web"}, {Name: " web-copy "}},
		Hosts:  []config.Host{{Host: "db"}, {Host: "web"}},
	}
	tc := []struct {
		name, got, want string
	}{
		{"group: trimmed existing copy is taken", suggestCopyGroupName(inv, "web"), "web-copy2"},
		{"group: free name", suggestCopyGroupName(inv, "db"), "db-copy"},
		{"host: trims the source", suggestCopyHostKey(inv, " db "), "db-copy"},
		{"host: group names do not count", suggestCopyHostKey(inv, "web-copy"), "web-copy-copy"},
		{"blank source", suggestCopyName(" ", func(string) bool { return false }), "copy-copy"},
		{"all taken falls back", suggestCopyName("x", func(string) bool { return true }), "x-copy"},
		{"gives up after copy999", suggestCopyName("x", func(n string) bool { return n != "x-copy1000" }), "x-copy"},
	}
	for _, c := range tc {
		if c.got != c.want {
			t.Errorf("%s: got %q want %q", c.name, c.got, c.want)
		}
	}
}
