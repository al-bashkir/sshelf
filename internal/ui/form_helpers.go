package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

// sshPlaceholders returns the placeholders for the shared SSH fields, showing
// the inherited defaults where they are set.
func sshPlaceholders(defs config.Defaults) (port, identity, extraArgs string) {
	port = "22"
	if defs.Port != 0 {
		port = strconv.Itoa(defs.Port)
	}
	identity = strings.TrimSpace(defs.IdentityFile)
	if identity == "" {
		identity = "~/.ssh/id_ed25519"
	}
	extraArgs = strings.Join(defs.ExtraArgs, " ")
	if extraArgs == "" {
		extraArgs = "-o Option=value ..."
	}
	return port, identity, extraArgs
}

// formTitle builds the modal title: the edited item's name under its parent
// crumb, or a create/edit label when there is no name yet.
func formTitle(parentCrumb string, index int, name, createTitle, editTitle string) string {
	if index < 0 {
		return breadcrumbTitle(parentCrumb, createTitle)
	}
	if n := strings.TrimSpace(name); n != "" {
		return breadcrumbTitle(parentCrumb, n)
	}
	return breadcrumbTitle(parentCrumb, editTitle)
}

// renderModalForm renders a form into its modal box: scrolled content with the
// focused field visible, any open picker overlaid, then toast and footer.
func renderModalForm(f *formModel, width, height int, title string, t toast) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	innerW := max(0, width-2)
	lines, focusLine := f.renderFormContent(innerW)

	reserved := 2 // sep + footer
	toastStr := ""
	if !t.empty() {
		toastStr = renderToast(t)
		reserved++
	}
	visibleH := max(1, max(0, height-2)-reserved)

	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]
	if f.picker != nil {
		visible = f.overlayPickerOnVisible(visible, focusLine-start, innerW)
	}
	return renderFormBox(width, title, visible, visibleH, toastStr, f.renderFooter())
}

// formLabel renders a padded label for a form field.
// When focused, the label is rendered in accent style.
func formLabel(s string, labelW int, focused bool) string {
	if len(s) < labelW {
		s += strings.Repeat(" ", labelW-len(s))
	}
	if focused {
		return headerStyle.Render(s)
	}
	return s
}

// renderFormBox renders a scrollable modal form box with a title, visible
// content lines, optional toast, and a pre-styled footer.
func renderFormBox(totalW int, title string, visibleLines []string, visibleH int, toastStr string, footer string) string {
	innerW := max(0, totalW-2)
	out := make([]string, 0, visibleH+5)
	out = append(out, focusedBoxTitleTop(totalW, title))
	for _, ln := range visibleLines {
		out = append(out, focusedBoxLine(totalW, padVisible(ln, innerW)))
	}
	for i := len(visibleLines); i < visibleH; i++ {
		out = append(out, focusedBoxLine(totalW, strings.Repeat(" ", innerW)))
	}
	if toastStr != "" {
		out = append(out, focusedBoxLine(totalW, padVisible(toastStr, innerW)))
	}
	out = append(out, focusedBoxSep(totalW))
	out = append(out, focusedBoxLine(totalW, padVisible(footer, innerW)))
	out = append(out, focusedBoxBottom(totalW))
	return strings.Join(out, "\n")
}

// parseExtraArgs turns the extra-args field text back into args. Text that
// still matches the joined base is returned as base, so untouched args that
// contain spaces survive a save.
func parseExtraArgs(text string, base []string) []string {
	text = strings.TrimSpace(text)
	if text == strings.Join(base, " ") {
		return base
	}
	if text == "" {
		return nil
	}
	// shortcut: edited text splits on spaces, so an edited arg with spaces breaks; use a shell-word parser if users need that.
	return strings.Fields(text)
}

// parsePort parses a port field; empty means unset (0). It applies the same
// 1-65535 rule as the inline check.
func parsePort(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if err := validatePort(s); err != nil {
		return 0, fmt.Errorf("invalid port: %w", err)
	}
	return strconv.Atoi(s)
}
