package ui

import (
	"strings"
	"time"
)

// toastLevel represents the severity of a toast notification.
type toastLevel int

const (
	toastInfo toastLevel = iota // neutral information
	toastOK                     // success
	toastWarn                   // warning / prompt
	toastErr                    // error
)

// toast holds a notification message with an associated severity level.
type toast struct {
	text  string
	level toastLevel
}

func (t toast) empty() bool {
	return strings.TrimSpace(t.text) == ""
}

// toastDuration returns the auto-dismiss delay for a given severity.
func toastDuration(l toastLevel) time.Duration {
	switch l {
	case toastOK, toastInfo:
		return 3 * time.Second
	case toastWarn:
		return 5 * time.Second
	case toastErr:
		return 8 * time.Second
	}
	return 4 * time.Second
}

// renderToast returns the styled string for a toast.
func renderToast(t toast) string { return renderToastWithSpinner(t, false) }

// renderToastWithSpinner returns the styled string for a toast, optionally
// prepended with a spinner frame.
func renderToastWithSpinner(t toast, spinner bool) string {
	if t.empty() {
		if spinner {
			return statusWarn.Render(spinnerFrame())
		}
		return ""
	}
	text := t.text
	if spinner {
		text = spinnerFrame() + " " + text
	}
	switch t.level {
	case toastOK:
		return toastOKStyle.Render(text)
	case toastInfo:
		return toastInfoStyle.Render(text)
	case toastErr:
		return toastErrStyle.Render(text)
	}
	return statusWarn.Render(text)
}

// breadcrumbTitle builds a breadcrumb-style title:
// "parentCrumb > leafTitle" where parent segments are dim and leaf is accented.
func breadcrumbTitle(parentCrumb, leafTitle string) string {
	parentCrumb = strings.TrimSpace(parentCrumb)
	leafTitle = strings.TrimSpace(leafTitle)
	if parentCrumb == "" {
		return leafTitle
	}
	return dim.Render(parentCrumb+" >") + " " + headerStyle.Render(leafTitle)
}
