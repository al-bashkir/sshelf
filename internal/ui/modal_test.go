package ui

import "testing"

func TestModalSizes(t *testing.T) {
	type size struct{ w, h int }
	sizers := map[string]func(int, int) (int, int){
		"group":  groupFormModalSize,
		"host":   hostFormModalSize,
		"picker": pickerModalSize,
		"custom": customHostModalSize,
	}
	tc := []struct {
		sizer      string
		full, want size
	}{
		// Unknown terminal size: the max size.
		{"group", size{0, 0}, size{110, 26}},
		{"custom", size{0, 0}, size{76, 9}},
		// Large terminal: capped at max.
		{"group", size{200, 60}, size{110, 26}},
		{"host", size{200, 60}, size{96, 20}},
		{"picker", size{200, 60}, size{90, 22}},
		{"custom", size{200, 60}, size{76, 9}},
		// Medium terminal: terminal minus margins.
		{"group", size{50, 20}, size{46, 19}},
		{"host", size{50, 20}, size{44, 16}},
		{"picker", size{50, 20}, size{44, 14}},
		// Too few rows left: full height (custom: at most 9).
		{"group", size{50, 16}, size{46, 16}},
		{"group", size{50, 17}, size{46, 16}}, // exactly minH rows left: keep
		{"custom", size{50, 18}, size{44, 8}}, // exactly 8 rows left: keep
		{"host", size{50, 15}, size{44, 15}},
		{"picker", size{50, 15}, size{44, 15}},
		{"custom", size{50, 17}, size{44, 9}},
		{"custom", size{50, 5}, size{44, 5}},
		// No room for margins: full width and height.
		{"group", size{3, 1}, size{3, 1}},
		{"host", size{6, 4}, size{6, 4}},
	}
	for _, c := range tc {
		w, h := sizers[c.sizer](c.full.w, c.full.h)
		if (size{w, h}) != c.want {
			t.Errorf("%s %v: got %v want %v", c.sizer, c.full, size{w, h}, c.want)
		}
	}
}

func TestModalSizeEdges(t *testing.T) {
	// No max: terminal minus margins; a negative margin cannot overflow.
	if w, h := modalSize(80, 30, 0, 0, 10, 5); w != 70 || h != 25 {
		t.Errorf("no max: got %d,%d want 70,25", w, h)
	}
	if w, h := modalSize(10, 10, 0, 0, -5, -5); w != 10 || h != 10 {
		t.Errorf("negative margin: got %d,%d want 10,10", w, h)
	}
	// Unknown size with no max: zero.
	if w, h := modalSize(0, -1, 0, 0, 4, 4); w != 0 || h != 0 {
		t.Errorf("unknown, no max: got %d,%d want 0,0", w, h)
	}
	// Smallest cap and smallest room left still count.
	if w, h := modalSize(80, 30, 1, 1, 0, 0); w != 1 || h != 1 {
		t.Errorf("max 1: got %d,%d want 1,1", w, h)
	}
	if w, h := modalSize(10, 10, 0, 0, 9, 9); w != 1 || h != 1 {
		t.Errorf("1 cell left: got %d,%d want 1,1", w, h)
	}
}

func TestPlaceCentered(t *testing.T) {
	tc := []struct {
		w, h      int
		box, want string
	}{
		{0, 5, "box\n", "box"}, // unknown width
		{5, 0, "box\n", "box"}, // unknown height
		{5, 3, "ab\n", "     \n ab  \n     "},
		{1, 3, "a", " \na\n "}, // one column still centers
		{3, 1, "a", " a "},     // one row still centers
	}
	for _, c := range tc {
		if got := placeCentered(c.w, c.h, c.box); got != c.want {
			t.Errorf("placeCentered(%d,%d,%q): got %q want %q", c.w, c.h, c.box, got, c.want)
		}
	}
}
