package tmux

import (
	"slices"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

// PaneSettings holds resolved tmux pane display options.
type PaneSettings struct {
	SplitFlag    string
	Layout       string
	SyncPanes    bool
	BorderFormat string
	BorderStatus string
}

// ResolvePaneSettings merges defaults and optional group overrides into PaneSettings.
func ResolvePaneSettings(defaults config.Defaults, group *config.Group, paneCount int) PaneSettings {
	var g config.Group
	if group != nil {
		g = *group
	}
	// A non-blank group value wins over the default.
	split := strings.ToLower(orDefault(g.PaneSplit, strings.TrimSpace(defaults.PaneSplit)))
	sync := strings.ToLower(orDefault(g.PaneSync, strings.TrimSpace(defaults.PaneSync)))

	// Anything that is not an explicit "vertical" splits horizontally.
	splitFlag := "-h"
	if split == "vertical" || split == "v" {
		splitFlag = "-v"
	}

	return PaneSettings{
		SplitFlag:    splitFlag,
		Layout:       layoutName(orDefault(g.PaneLayout, strings.TrimSpace(defaults.PaneLayout)), paneCount),
		SyncPanes:    !slices.Contains([]string{"off", "false", "0", "no"}, sync), // on unless disabled
		BorderFormat: orDefault(g.PaneBorderFmt, strings.TrimSpace(defaults.PaneBorderFmt)),
		BorderStatus: orDefault(g.PaneBorderPos, strings.TrimSpace(defaults.PaneBorderPos)),
	}
}

var layoutAliases = map[string]string{
	"t": "tiled", "tiled": "tiled",
	"eh": "even-horizontal", "even-horizontal": "even-horizontal",
	"ev": "even-vertical", "even-vertical": "even-vertical",
	"mh": "main-horizontal", "main-horizontal": "main-horizontal",
	"mv": "main-vertical", "main-vertical": "main-vertical",
}

// layoutName maps a trimmed layout setting to a tmux layout. Blank or "auto"
// picks by pane count; an unknown name passes through unchanged.
func layoutName(layout string, paneCount int) string {
	key := strings.ToLower(layout)
	if key == "" || key == "auto" {
		if paneCount >= 4 {
			return "tiled"
		}
		return "even-vertical"
	}
	if name, ok := layoutAliases[key]; ok {
		return name
	}
	return layout
}
