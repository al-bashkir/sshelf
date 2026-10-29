package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Modal size constants.
const (
	modalMaxWGroup  = 110
	modalMaxHGroup  = 26
	modalMaxWHost   = 96
	modalMaxHHost   = 20
	modalMaxWPicker = 90
	modalMaxHPicker = 22
	modalMaxWCustom = 76
	modalMaxHCustom = 9
)

func placeCentered(fullW, fullH int, box string) string {
	box = strings.TrimRight(box, "\n")
	if fullW <= 0 || fullH <= 0 {
		return box
	}
	return lipgloss.Place(fullW, fullH, lipgloss.Center, lipgloss.Center, box)
}

func modalSize(fullW, fullH int, maxW, maxH int, marginW, marginH int) (w, h int) {
	// width
	w = maxW
	if fullW > 0 {
		w = fullW - marginW
		if w < 0 {
			w = 0
		}
	}
	if maxW > 0 && w > maxW {
		w = maxW
	}
	if fullW > 0 {
		if w <= 0 {
			w = fullW
		}
		if w > fullW {
			w = fullW
		}
	}

	// height
	h = maxH
	if fullH > 0 {
		h = fullH - marginH
		if h < 0 {
			h = 0
		}
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}
	if fullH > 0 {
		if h <= 0 {
			h = fullH
		}
		if h > fullH {
			h = fullH
		}
	}

	return w, h
}

func groupFormModalSize(fullW, fullH int) (w, h int) {
	// Big form: keep almost full height if needed.
	w, h = modalSize(fullW, fullH, modalMaxWGroup, modalMaxHGroup, 4, 1)
	if fullH > 0 && h < 16 {
		// Fallback to full height in very small terminals.
		h = fullH
	}
	return w, h
}

func hostFormModalSize(fullW, fullH int) (w, h int) {
	w, h = modalSize(fullW, fullH, modalMaxWHost, modalMaxHHost, 6, 4)
	if fullH > 0 && h < 12 {
		h = fullH
	}
	return w, h
}

func pickerModalSize(fullW, fullH int) (w, h int) {
	w, h = modalSize(fullW, fullH, modalMaxWPicker, modalMaxHPicker, 6, 6)
	if fullH > 0 && h < 10 {
		h = fullH
	}
	return w, h
}

func customHostModalSize(fullW, fullH int) (w, h int) {
	w, h = modalSize(fullW, fullH, modalMaxWCustom, modalMaxHCustom, 6, 10)
	if fullH > 0 && h < 8 {
		h = min(9, fullH)
	}
	return w, h
}
