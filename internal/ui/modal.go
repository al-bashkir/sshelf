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

// modalSize fits a modal into a fullW x fullH terminal (0 or less: unknown).
func modalSize(fullW, fullH int, maxW, maxH int, marginW, marginH int) (w, h int) {
	return fitAxis(fullW, maxW, marginW), fitAxis(fullH, maxH, marginH)
}

// fitAxis sizes one axis: the terminal minus the margin, capped at limit
// (0 or less: no cap). With no room for the margin, it takes the full axis.
// An unknown terminal size gives limit.
func fitAxis(full, limit, margin int) int {
	if full <= 0 {
		return limit
	}
	v := full - margin
	if limit > 0 {
		v = min(v, limit)
	}
	if v <= 0 {
		return full
	}
	return min(v, full)
}

// fitModal is modalSize that falls back to the full height when fewer than
// minH rows are left, for very small terminals.
func fitModal(fullW, fullH, maxW, maxH, marginW, marginH, minH int) (w, h int) {
	w, h = modalSize(fullW, fullH, maxW, maxH, marginW, marginH)
	if fullH > 0 && h < minH {
		h = fullH
	}
	return w, h
}

func groupFormModalSize(fullW, fullH int) (w, h int) {
	// Big form: keep almost full height if needed.
	return fitModal(fullW, fullH, modalMaxWGroup, modalMaxHGroup, 4, 1, 16)
}

func hostFormModalSize(fullW, fullH int) (w, h int) {
	return fitModal(fullW, fullH, modalMaxWHost, modalMaxHHost, 6, 4, 12)
}

func pickerModalSize(fullW, fullH int) (w, h int) {
	return fitModal(fullW, fullH, modalMaxWPicker, modalMaxHPicker, 6, 6, 10)
}

func customHostModalSize(fullW, fullH int) (w, h int) {
	w, h = modalSize(fullW, fullH, modalMaxWCustom, modalMaxHCustom, 6, 10)
	if fullH > 0 && h < 8 {
		h = min(9, fullH)
	}
	return w, h
}
