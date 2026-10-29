package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingReturnsDefaults(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "missing.toml")

	cfg, used, err := Load(p)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("cfg=%#v, want defaults", cfg)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "config.toml")

	cfg := DefaultConfig()
	cfg.Defaults.User = "me"

	if _, err := Save(p, &cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o, want 600", st.Mode().Perm())
	}

	got, used, err := Load(p)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}

	cfg.Version = 1
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("got=%#v\nwant=%#v", got, cfg)
	}
}

func TestLoadInventoryMissingReturnsDefaults(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "missing.toml")

	inv, used, err := LoadInventory(p)
	if err != nil {
		t.Fatalf("LoadInventory error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}
	if !reflect.DeepEqual(inv, DefaultInventory()) {
		t.Fatalf("inv=%#v, want defaults", inv)
	}
}

func TestSaveInventoryThenLoadRoundTrip(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "hosts.toml")

	inv := DefaultInventory()
	inv.Hosts = []Host{{
		Host:         "db1.example",
		User:         "ubuntu",
		Port:         2222,
		IdentityFile: "~/.ssh/db1_ed25519",
		ExtraArgs:    []string{"-o", "ServerAliveInterval=30"},
	}}
	inv.Groups = []Group{{
		Name:         "prod",
		User:         "admin",
		Port:         22,
		IdentityFile: "~/.ssh/prod_ed25519",
		ExtraArgs:    []string{"-o", "ServerAliveInterval=30"},
		OpenMode:     "tmux-window",
		Hosts:        []string{"db1.example", "[10.0.0.1]:2222"},
	}}

	if _, err := SaveInventory(p, &inv); err != nil {
		t.Fatalf("SaveInventory error: %v", err)
	}

	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o, want 600", st.Mode().Perm())
	}

	got, used, err := LoadInventory(p)
	if err != nil {
		t.Fatalf("LoadInventory error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}

	inv.Version = 1
	if !reflect.DeepEqual(got, inv) {
		t.Fatalf("got=%#v\nwant=%#v", got, inv)
	}
}

func TestMigrateOldConfig(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "config.toml")
	hostsPath := filepath.Join(d, "hosts.toml")

	// Write an old-format config.toml that has both defaults and hosts/groups.
	oldContent := `version = 1
hidden_hosts = ["hidden.example"]

[defaults]
user = "me"
port = 22

[[hosts]]
host = "db1.example"
user = "ubuntu"
port = 2222

[[groups]]
name = "prod"
hosts = ["db1.example"]
`
	if err := os.WriteFile(cfgPath, []byte(oldContent), 0o600); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	// Run migration.
	if err := Migrate(cfgPath, hostsPath); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}

	// Verify hosts.toml was created with the host/group data.
	inv, _, err := LoadInventory(hostsPath)
	if err != nil {
		t.Fatalf("LoadInventory after migrate: %v", err)
	}
	if len(inv.Hosts) != 1 {
		t.Fatalf("inv.Hosts=%d, want 1", len(inv.Hosts))
	}
	if inv.Hosts[0].Host != "db1.example" {
		t.Fatalf("inv.Hosts[0].Host=%q, want db1.example", inv.Hosts[0].Host)
	}
	if len(inv.Groups) != 1 {
		t.Fatalf("inv.Groups=%d, want 1", len(inv.Groups))
	}
	if inv.Groups[0].Name != "prod" {
		t.Fatalf("inv.Groups[0].Name=%q, want prod", inv.Groups[0].Name)
	}
	if len(inv.HiddenHosts) != 1 || inv.HiddenHosts[0] != "hidden.example" {
		t.Fatalf("inv.HiddenHosts=%v, want [hidden.example]", inv.HiddenHosts)
	}

	// Verify config.toml was rewritten without host/group data.
	cfg, _, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load after migrate: %v", err)
	}
	if cfg.Defaults.User != "me" {
		t.Fatalf("cfg.Defaults.User=%q, want me", cfg.Defaults.User)
	}

	// Verify TOML file no longer contains [[hosts]] or [[groups]] sections.
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read migrated config: %v", err)
	}
	content := string(raw)
	for _, s := range []string{"[[hosts]]", "[[groups]]", "hidden_hosts"} {
		if strings.Contains(content, s) {
			t.Fatalf("migrated config.toml still contains %q", s)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "config.toml")
	hostsPath := filepath.Join(d, "hosts.toml")

	// Write config and hosts files.
	cfg := DefaultConfig()
	cfg.Defaults.User = "me"
	if _, err := Save(cfgPath, &cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	inv := DefaultInventory()
	inv.Groups = []Group{{Name: "test", Hosts: []string{"h1"}}}
	if _, err := SaveInventory(hostsPath, &inv); err != nil {
		t.Fatalf("SaveInventory: %v", err)
	}

	// Run migration — should be a no-op since hosts.toml exists.
	if err := Migrate(cfgPath, hostsPath); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}

	// Verify hosts.toml is unchanged.
	got, _, err := LoadInventory(hostsPath)
	if err != nil {
		t.Fatalf("LoadInventory: %v", err)
	}
	inv.Version = 1
	if !reflect.DeepEqual(got, inv) {
		t.Fatalf("got=%#v\nwant=%#v", got, inv)
	}
}

func TestMigrateNoHostData(t *testing.T) {
	d := t.TempDir()
	cfgPath := filepath.Join(d, "config.toml")
	hostsPath := filepath.Join(d, "hosts.toml")

	// Write a config with no host/group data.
	cfg := DefaultConfig()
	cfg.Defaults.User = "me"
	if _, err := Save(cfgPath, &cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Migration should not create hosts.toml since there's nothing to migrate.
	if err := Migrate(cfgPath, hostsPath); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}
	if _, err := os.Stat(hostsPath); !os.IsNotExist(err) {
		t.Fatalf("hosts.toml should not exist after migration with no host data")
	}
}
