package ui

import (
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

func paneBorderFormatChoices(defs config.Defaults) []string {
	out := []string{config.DefaultPaneBorderFormat}
	seen := map[string]bool{strings.TrimSpace(config.DefaultPaneBorderFormat): true}

	for _, v := range defs.PaneBorderFmts {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if seen[v] {
			continue
		}
		out = append(out, v)
		seen[v] = true
	}

	cur := strings.TrimSpace(defs.PaneBorderFmt)
	if cur != "" && !seen[cur] {
		out = append(out, cur)
	}
	return out
}

func removePaneBorderFormat(defs *config.Defaults, v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return
	}
	if strings.TrimSpace(config.DefaultPaneBorderFormat) == v {
		return
	}
	if len(defs.PaneBorderFmts) == 0 {
		return
	}

	out := defs.PaneBorderFmts[:0]
	for _, s := range defs.PaneBorderFmts {
		ss := strings.TrimSpace(s)
		if ss == "" {
			continue
		}
		if ss == v {
			continue
		}
		out = append(out, s)
	}
	defs.PaneBorderFmts = out
}

func addPaneBorderFormat(defs *config.Defaults, v string) (added bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	if strings.TrimSpace(config.DefaultPaneBorderFormat) == v {
		return false
	}
	for _, s := range defs.PaneBorderFmts {
		if strings.TrimSpace(s) == v {
			return false
		}
	}
	defs.PaneBorderFmts = append(defs.PaneBorderFmts, v)
	return true
}
