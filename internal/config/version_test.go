package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadTreatsVersionZeroAsOne(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte("version = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(p)
	if err != nil || cfg.Version != 1 {
		t.Fatalf("version=%d err=%v, want 1", cfg.Version, err)
	}
}

func TestLoadInventoryTreatsVersionZeroAsOne(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hosts.toml")
	if err := os.WriteFile(p, []byte("version = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inv, _, err := LoadInventory(p)
	if err != nil || inv.Version != 1 {
		t.Fatalf("version=%d err=%v, want 1", inv.Version, err)
	}
}

func TestSaveWritesVersionOne(t *testing.T) {
	d := t.TempDir()
	cp, hp := filepath.Join(d, "config.toml"), filepath.Join(d, "hosts.toml")
	if _, err := Save(cp, &Config{}); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveInventory(hp, &Inventory{}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{cp, hp} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "version = 1") {
			t.Errorf("%s lacks version = 1:\n%s", p, data)
		}
	}
}

func TestSaveEmptyPathUsesDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := DefaultConfig()
	used, err := Save("", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := DefaultPath()
	if used != want {
		t.Fatalf("used=%q want %q", used, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfigDocumentedDefaults(t *testing.T) {
	d := DefaultConfig().Defaults
	if !d.LoadKnownHosts || d.ConfirmQuit || !d.ShowFieldHelp {
		t.Fatalf("load_known_hosts=%v confirm_quit=%v show_field_help=%v, want true/false/true",
			d.LoadKnownHosts, d.ConfirmQuit, d.ShowFieldHelp)
	}
}

func TestMigrateAnyLegacyDataCreatesHosts(t *testing.T) {
	for name, legacy := range map[string]string{
		"hosts+groups": "[[hosts]]\nhost = \"a\"\n\n[[groups]]\nname = \"g\"\n",
		"hosts+hidden": "hidden_hosts = [\"h\"]\n\n[[hosts]]\nhost = \"a\"\n",
		"hidden only":  "hidden_hosts = [\"h\"]\n",
		"groups only":  "[[groups]]\nname = \"g\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			d := t.TempDir()
			cp, hp := filepath.Join(d, "config.toml"), filepath.Join(d, "hosts.toml")
			if err := os.WriteFile(cp, []byte(legacy), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Migrate(cp, hp); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(hp); err != nil {
				t.Fatalf("hosts.toml not created: %v", err)
			}
			data, _ := os.ReadFile(hp)
			if !strings.Contains(string(data), "version = 1") {
				t.Fatalf("hosts.toml lacks version:\n%s", data)
			}
			cdata, _ := os.ReadFile(cp)
			if !strings.Contains(string(cdata), "version = 1") {
				t.Fatalf("config.toml lacks version:\n%s", cdata)
			}
		})
	}
}

func TestConfigHosts(t *testing.T) {
	inv := Inventory{
		Hosts:  []Host{{Host: " b "}, {Host: ""}, {Host: "  "}, {Host: "a"}},
		Groups: []Group{{Hosts: []string{"c", " a ", "", "b"}}, {Hosts: []string{"  "}}},
	}
	got := ConfigHosts(inv)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%q want %q", got, want)
	}
	if got := ConfigHosts(Inventory{Groups: []Group{{Hosts: []string{"z"}}}}); !reflect.DeepEqual(got, []string{"z"}) {
		t.Fatalf("group-only: got=%q", got)
	}
}
