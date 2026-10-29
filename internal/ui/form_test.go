package ui

import (
	"slices"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

var proxyArgs = []string{"-o", "ProxyCommand=ssh -W %h:%p bastion"}

func TestHostFormKeepsExtraArgsWithSpaces(t *testing.T) {
	base := config.Host{Host: "box", User: "old", ExtraArgs: proxyArgs}
	v := hostToValues(base)
	v["user"] = "new"
	h, err := applyHostValues(v, base)
	if err != nil {
		t.Fatal(err)
	}
	if h.User != "new" || !slices.Equal(h.ExtraArgs, proxyArgs) {
		t.Fatalf("got user=%q args=%q", h.User, h.ExtraArgs)
	}
}

func TestGroupFormKeepsExtraArgsWithSpaces(t *testing.T) {
	base := config.Group{Name: "prod", User: "old", ExtraArgs: proxyArgs}
	v := groupToValues(base)
	v["user"] = "new"
	g, err := applyGroupValues(v, base)
	if err != nil {
		t.Fatal(err)
	}
	if g.User != "new" || !slices.Equal(g.ExtraArgs, proxyArgs) {
		t.Fatalf("got user=%q args=%q", g.User, g.ExtraArgs)
	}
}

func TestDefaultsFormKeepsExtraArgsWithSpaces(t *testing.T) {
	base := config.DefaultConfig().Defaults
	base.ExtraArgs = proxyArgs
	v := defaultsToValues(base)
	v["user"] = "new"
	d, err := applySettingsValues(v, base)
	if err != nil {
		t.Fatal(err)
	}
	if d.User != "new" || !slices.Equal(d.ExtraArgs, proxyArgs) {
		t.Fatalf("got user=%q args=%q", d.User, d.ExtraArgs)
	}
}

func TestFormsRejectOutOfRangePort(t *testing.T) {
	if _, err := applyHostValues(map[string]string{"host": "box", "port": "70000"}, config.Host{}); err == nil {
		t.Error("host form accepted port 70000")
	}
	if _, err := applyGroupValues(map[string]string{"name": "prod", "port": "70000"}, config.Group{}); err == nil {
		t.Error("group form accepted port 70000")
	}
	base := config.DefaultConfig().Defaults
	v := defaultsToValues(base)
	v["port"] = "70000"
	if _, err := applySettingsValues(v, base); err == nil {
		t.Error("settings form accepted port 70000")
	}
}

// --- unsaved-changes guard ---

func keyOf(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "alt+1":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1"), Alt: true}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// press sends one key and returns the message its command produces, if any.
func press(m tea.Model, s string) tea.Msg {
	_, cmd := m.Update(keyOf(s))
	if cmd == nil {
		return nil
	}
	return cmd()
}

// dirty edits the first field the way a user does: i, a letter, Esc.
func dirty(t *testing.T, m tea.Model) {
	t.Helper()
	press(m, "i")
	press(m, "x")
	if press(m, "esc") != nil {
		t.Fatal("leaving insert mode emitted a message")
	}
}

type formCase struct {
	name   string
	new    func() tea.Model
	cancel func(tea.Msg) bool
}

func formCases() []formCase {
	defs := config.DefaultConfig().Defaults
	return []formCase{
		{"host", func() tea.Model { return newHostFormModel(0, config.Host{Host: "box"}, defs) },
			func(m tea.Msg) bool { _, ok := m.(hostFormCancelMsg); return ok }},
		{"group", func() tea.Model { return newGroupFormModel(0, config.Group{Name: "prod"}, defs) },
			func(m tea.Msg) bool { _, ok := m.(groupFormCancelMsg); return ok }},
		{"settings", func() tea.Model { return newDefaultsFormModel(defs, false) },
			func(m tea.Msg) bool { _, ok := m.(defaultsFormCancelMsg); return ok }},
	}
}

func TestDirtyFormAsksBeforeEsc(t *testing.T) {
	for _, c := range formCases() {
		t.Run(c.name, func(t *testing.T) {
			m := c.new()
			dirty(t, m)
			if got := press(m, "esc"); got != nil {
				t.Fatalf("dirty form emitted %T on Esc, want confirm", got)
			}
			if got := press(m, "n"); got != nil {
				t.Fatalf("n emitted %T", got)
			}
			if got := press(m, "esc"); got != nil {
				t.Fatalf("form lost its edits after n: Esc emitted %T", got)
			}
			if got := press(m, "y"); !c.cancel(got) {
				t.Fatalf("y emitted %T, want cancel", got)
			}
		})
	}
}

func TestCleanFormEscExitsAtOnce(t *testing.T) {
	for _, c := range formCases() {
		t.Run(c.name, func(t *testing.T) {
			if got := press(c.new(), "esc"); !c.cancel(got) {
				t.Fatalf("clean Esc emitted %T, want cancel", got)
			}
		})
	}
}

func TestDirtySettingsAsksBeforeQuitAndTabSwitch(t *testing.T) {
	defs := config.DefaultConfig().Defaults
	for _, k := range []string{"q", "alt+1"} {
		m := newDefaultsFormModel(defs, false)
		dirty(t, m)
		if got := press(m, k); got != nil {
			t.Fatalf("dirty settings emitted %T on %s, want confirm", got, k)
		}
		got := press(m, "y")
		switch k {
		case "q":
			if _, ok := got.(tea.QuitMsg); !ok {
				t.Fatalf("y after q emitted %T, want QuitMsg", got)
			}
		default:
			if s, ok := got.(switchScreenMsg); !ok || s.to != screenHosts {
				t.Fatalf("y after %s emitted %#v, want switch to hosts", k, got)
			}
		}
	}
	// A clean form quits at once.
	if _, ok := press(newDefaultsFormModel(defs, false), "q").(tea.QuitMsg); !ok {
		t.Fatal("clean settings did not quit on q")
	}
}
