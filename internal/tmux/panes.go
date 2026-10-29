package tmux

import (
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
	split := strings.TrimSpace(defaults.PaneSplit)
	layout := strings.TrimSpace(defaults.PaneLayout)
	sync := strings.TrimSpace(defaults.PaneSync)
	borderFmt := strings.TrimSpace(defaults.PaneBorderFmt)
	borderPos := strings.TrimSpace(defaults.PaneBorderPos)

	if group != nil {
		if v := strings.TrimSpace(group.PaneSplit); v != "" {
			split = v
		}
		if v := strings.TrimSpace(group.PaneLayout); v != "" {
			layout = v
		}
		if v := strings.TrimSpace(group.PaneSync); v != "" {
			sync = v
		}
		if v := strings.TrimSpace(group.PaneBorderFmt); v != "" {
			borderFmt = v
		}
		if v := strings.TrimSpace(group.PaneBorderPos); v != "" {
			borderPos = v
		}
	}

	// Anything that is not an explicit "vertical" splits horizontally.
	splitFlag := "-h"
	if v := strings.ToLower(strings.TrimSpace(split)); v == "vertical" || v == "v" {
		splitFlag = "-v"
	}

	layoutName := ""
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "", "auto":
		if paneCount >= 4 {
			layoutName = "tiled"
		} else {
			layoutName = "even-vertical"
		}
	case "t", "tiled":
		layoutName = "tiled"
	case "eh", "even-horizontal":
		layoutName = "even-horizontal"
	case "ev", "even-vertical":
		layoutName = "even-vertical"
	case "mh", "main-horizontal":
		layoutName = "main-horizontal"
	case "mv", "main-vertical":
		layoutName = "main-vertical"
	default:
		layoutName = strings.TrimSpace(layout)
	}

	// Sync is on unless explicitly disabled.
	syncOn := true
	switch strings.ToLower(strings.TrimSpace(sync)) {
	case "off", "false", "0", "no":
		syncOn = false
	}

	return PaneSettings{
		SplitFlag:    splitFlag,
		Layout:       layoutName,
		SyncPanes:    syncOn,
		BorderFormat: borderFmt,
		BorderStatus: borderPos,
	}
}
