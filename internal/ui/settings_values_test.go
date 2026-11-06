package ui

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

// base has custom border formats with blanks, duplicates and the built-in
// default; every save keeps only the unique custom ones.
func settingsBase() config.Defaults {
	return config.Defaults{
		ExtraArgs:      []string{"-A"},
		PaneBorderFmts: []string{" a ", "", "a", config.DefaultPaneBorderFormat, "b"},
	}
}

func TestApplySettingsValuesBlankForm(t *testing.T) {
	got, err := applySettingsValues(map[string]string{}, settingsBase())
	if err != nil {
		t.Fatal(err)
	}
	want := config.Defaults{
		LoadKnownHosts:          true,
		ShowFieldHelp:           true,
		Port:                    22,
		TmuxSession:             "sshelf",
		ConnectConfirmThreshold: 5,
		PaneBorderFmt:           config.DefaultPaneBorderFormat,
		PaneBorderFmts:          []string{"a", "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestApplySettingsValuesFilledForm(t *testing.T) {
	values := map[string]string{
		"user": " u ", "identity": " ~/.ssh/k ", "colorscheme": " Nord ", "accent": " DEFAULT ",
		"load_known_hosts": "no", "confirm_quit": "yes", "show_field_help": "no",
		"port": " 2222 ", "extra_args": " -A ",
		"tmux": " force ", "open_mode": " tmux-pane ", "session": " s ", "threshold": " 0 ",
		"pane_split": " v ", "pane_layout": " tiled ", "pane_sync": " off ",
		"border_pos": " top ", "border_fmt": " #{pane_title} ",
	}
	got, err := applySettingsValues(values, settingsBase())
	if err != nil {
		t.Fatal(err)
	}
	want := config.Defaults{
		User: "u", IdentityFile: "~/.ssh/k", Colorscheme: "nord", AccentColor: "",
		LoadKnownHosts: false, ConfirmQuit: true, ShowFieldHelp: false,
		Port: 2222, ExtraArgs: []string{"-A"},
		Tmux: "force", OpenMode: "tmux-pane", TmuxSession: "s", ConnectConfirmThreshold: 0,
		PaneSplit: "v", PaneLayout: "tiled", PaneSync: "off",
		PaneBorderPos: "top", PaneBorderFmt: "#{pane_title}",
		PaneBorderFmts: []string{"a", "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestApplySettingsValuesColors(t *testing.T) {
	got, err := applySettingsValues(map[string]string{"colorscheme": "Default", "accent": " Red "}, settingsBase())
	if err != nil || got.Colorscheme != "" || got.AccentColor != "red" {
		t.Fatalf("colorscheme=%q accent=%q err=%v, want \"\" red nil", got.Colorscheme, got.AccentColor, err)
	}
}

func TestApplySettingsValuesErrors(t *testing.T) {
	for _, values := range []map[string]string{
		{"port": "0"}, {"port": "x"}, {"threshold": "-1"}, {"threshold": "x"},
	} {
		if _, err := applySettingsValues(values, settingsBase()); err == nil {
			t.Errorf("%v: want an error", values)
		}
	}
}

func TestDefaultsToValuesLeavesUnsetNumbersBlank(t *testing.T) {
	for _, c := range []struct {
		port, threshold     int
		wantPort, wantThres string
	}{
		{0, -1, "", ""}, // unset: the form shows the default placeholders
		{1, 0, "1", "0"},
		{2222, 7, "2222", "7"},
	} {
		v := defaultsToValues(config.Defaults{Port: c.port, ConnectConfirmThreshold: c.threshold})
		if v["port"] != c.wantPort || v["threshold"] != c.wantThres {
			t.Errorf("port %d threshold %d: got %q and %q, want %q and %q",
				c.port, c.threshold, v["port"], v["threshold"], c.wantPort, c.wantThres)
		}
	}
}
