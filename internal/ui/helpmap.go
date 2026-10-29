package ui

import "github.com/charmbracelet/bubbles/key"

// helpSection is a named group of key bindings shown in the help modal.
type helpSection struct {
	title string
	keys  []key.Binding
}

type helpMap struct {
	sections []helpSection // named sections for grouped rendering
}
