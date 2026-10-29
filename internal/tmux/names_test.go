package tmux

import (
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestWindowName(t *testing.T) {
	tests := map[string]string{
		"":                      "ssh",
		"  ":                    "ssh",
		"[]":                    "ssh",
		" web01 ":               "web01",
		"[web01]:2222":          "web01_2222",
		"fe80::1":               "fe80__1",
		strings.Repeat("a", 30): strings.Repeat("a", 30),
		strings.Repeat("a", 31): strings.Repeat("a", 30),
	}
	for in, want := range tests {
		if got := WindowName(in); got != want {
			t.Errorf("WindowName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestGroupWindowName(t *testing.T) {
	tests := []struct {
		name  string
		hosts []string
		group *config.Group
		want  string
	}{
		{"group name wins", []string{"h1"}, &config.Group{Name: " prod "}, "prod"},
		{"blank group name", []string{"h1", "h2"}, &config.Group{Name: " "}, "h1"},
		{"nil group", []string{"[h1]:22"}, nil, "h1_22"},
		{"no hosts", nil, nil, "ssh"},
		{"no hosts, blank group", nil, &config.Group{}, "ssh"},
	}
	for _, tc := range tests {
		if got := GroupWindowName(tc.hosts, tc.group); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
