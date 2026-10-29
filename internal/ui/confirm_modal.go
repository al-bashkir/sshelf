package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderQuitConfirm(width, height int) string {
	box := quitConfirmBox(width)
	if width <= 0 || height <= 0 {
		return strings.TrimSpace(box)
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderConfirmBox builds a confirm dialog using manual box primitives.
// title is embedded in the top border; body and footer are indented 2 spaces.
func renderConfirmBox(totalW int, title, body, footer string) string {
	return strings.Join([]string{
		focusedBoxTitleTop(totalW, title),
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+body),
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+footer),
		focusedBoxLine(totalW, ""),
		focusedBoxBottom(totalW),
	}, "\n")
}

func quitConfirmBox(maxWidth int) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = quitConfirmMaxW
	}
	boxW = min(quitConfirmMaxW, max(22, boxW-confirmDialogMargin))
	return renderConfirmBox(boxW+confirmDialogPadding,
		confirmTitleStyle.Render("Quit?"),
		"Exit sshelf?",
		styledFooter("y/⏎ quit  n/Esc cancel"))
}

func deleteGroupConfirmBox(maxWidth int, name string, hostCount int) string {
	body := "This will remove the group"
	if name = strings.TrimSpace(name); name != "" {
		body = fmt.Sprintf("Delete %q (%d)?", name, hostCount)
	}
	return renderConfirmBox(confirmDialogWidth(maxWidth),
		confirmTitleStyle.Render("Delete group?"),
		body,
		styledFooter("y/⏎ delete  n/Esc cancel"))
}

// confirmDialogWidth clamps the dialog to the terminal width and adds the
// border/padding overhead, returning the total box width.
func confirmDialogWidth(maxWidth int) int {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = confirmDialogMaxW
	}
	return min(confirmDialogMaxW, max(24, boxW-confirmDialogMargin)) + confirmDialogPadding
}

// renderHostListConfirmBox is a confirm dialog listing up to 4 hosts, with a
// "+N more" trailer beyond that and an optional note line below the list.
func renderHostListConfirmBox(maxWidth int, title, footer string, hosts []string, note string) string {
	totalW := confirmDialogWidth(maxWidth)

	parts := []string{focusedBoxTitleTop(totalW, title), focusedBoxLine(totalW, "")}
	shown, extra := hosts, 0
	if len(hosts) > 4 {
		shown, extra = hosts[:4], len(hosts)-4
	}
	for i, h := range shown {
		line := "  " + h
		if i == len(shown)-1 && extra > 0 {
			line += fmt.Sprintf("    +%d more", extra)
		}
		parts = append(parts, focusedBoxLine(totalW, line))
	}
	if note != "" {
		parts = append(parts, focusedBoxLine(totalW, ""), focusedBoxLine(totalW, note))
	}
	parts = append(parts,
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+footer),
		focusedBoxLine(totalW, ""),
		focusedBoxBottom(totalW),
	)
	return strings.Join(parts, "\n")
}

func connectConfirmBox(maxWidth int, count int, hostNames []string) string {
	return renderHostListConfirmBox(maxWidth,
		confirmTitleStyle.Render(fmt.Sprintf("Connect %d hosts?", count)),
		styledFooter("y/⏎ connect  n/Esc cancel"),
		hostNames, "")
}

func removeHostsConfirmBox(maxWidth int, hosts []string, groupName string) string {
	note := ""
	if groupName != "" {
		note = dim.Render("  from ") + groupName
	}
	return renderHostListConfirmBox(maxWidth,
		confirmTitleStyle.Render("Remove hosts?"),
		styledFooter("y/⏎ remove  n/Esc cancel"),
		hosts, note)
}
