package ui

import (
	"fmt"
)

func (m *hostsModel) View() string {
	if v, ok := m.overlayView("Hosts", m.helpKeys, "Connect and run a remote command (keeps session open)."); ok {
		return v
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	right := m.status()
	return renderMainTabBoxWithFooter(m.width, m.height, 0, searchHeader(&m.search, m.width, right), right,
		m.listOrEmpty(m.emptyStateView), m.footerHints())
}

// status is the header's right side: a toast, the reload spinner, or the
// load state, the selection and the host counts.
func (m *hostsModel) status() string {
	switch {
	case !m.toast.empty():
		return renderToastWithSpinner(m.toast, spinnerActive)
	case spinnerActive:
		return statusWarn.Render(spinnerFrame())
	}
	hasWarn := m.opts.SkippedLines > 0 || len(m.opts.LoadErrors) > 0
	right := statusDot(true, hasWarn)
	if hasWarn {
		right += dim.Render(" load warnings")
	}
	if n := len(m.selected); n > 0 {
		right += "  " + badgeSelStyle.Render(fmt.Sprintf("%d selected", n))
	}
	total := len(m.allHosts)
	if !m.showHidden {
		total -= m.hiddenCount()
	}
	return right + m.hostCount(total) + m.hiddenNote()
}

// hiddenNote counts the hidden hosts: "N hidden", or "N showed" while
// they show.
func (m *hostsModel) hiddenNote() string {
	n := m.hiddenCount()
	switch {
	case n == 0:
		return ""
	case m.showHidden:
		return "  " + headerStyle.Render(fmt.Sprintf("%d showed", n))
	}
	return "  " + dim.Render(fmt.Sprintf("%d hidden", n))
}

// footerHints offers the selection keys while hosts are selected.
func (m *hostsModel) footerHints() string {
	if len(m.selected) > 0 {
		return m.hintFooter("\u21b5 connect  o panes  \u2423 clear  ? help",
			"\u21b5 connect  o panes  ·  Ctrl+o cmd  a add-to-group  ·  \u2423 clear",
			"e config  r reload  ·  Alt+2 groups  Alt+3 settings  ? help")
	}
	return m.hintFooter("\u21b5 connect  \u2423 select  ? help",
		"\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  c custom  Alt+2 groups  Ctrl+H hide",
		"e config  Ctrl+o cmd  a add  r reload  ·  tab search  H show hidden  ? help")
}

func (m *hostsModel) emptyStateView() string {
	var defaultMsg string
	if len(m.allHosts) == 0 {
		divider := formSection("", 26)
		hint1 := footerKeyStyle.Render("c") + hintStyle.Render("          custom host")
		hint3 := footerKeyStyle.Render("Alt+3") + hintStyle.Render("      open settings")
		if m.opts.Config.Defaults.LoadKnownHosts {
			hint2 := footerKeyStyle.Render("r") + hintStyle.Render("          reload known_hosts")
			defaultMsg = dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint2 + "\n" + hint3
		} else {
			note := dim.Render("(known_hosts loading is off)")
			defaultMsg = dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint3 + "\n" + note
		}
	} else {
		defaultMsg = dim.Render("No hosts.")
	}
	return renderListEmptyState(m.width, m.height, m.search.Value(), defaultMsg)
}
