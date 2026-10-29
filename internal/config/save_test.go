package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveWritesThroughSymlink(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "dotfiles", "hosts.toml")
	link := filepath.Join(d, "hosts.toml")
	inv := DefaultInventory()
	if _, err := SaveInventory(target, &inv); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	inv, _, err := LoadInventory(link)
	if err != nil {
		t.Fatal(err)
	}
	inv.HiddenHosts = []string{"web"}
	if _, err := SaveInventory(link, &inv); err != nil {
		t.Fatal(err)
	}

	if st, err := os.Lstat(link); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced by a regular file (err=%v)", err)
	}
	got, _, err := LoadInventory(target)
	if err != nil || !reflect.DeepEqual(got.HiddenHosts, []string{"web"}) {
		t.Fatalf("target hidden_hosts=%v err=%v, want [web]", got.HiddenHosts, err)
	}
}

func TestSaveInventoryRefusesFileChangedOnDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hosts.toml")
	// Two sshelf instances read the same (missing) file.
	a, _, _ := LoadInventory(p)
	b, _, _ := LoadInventory(p)

	b.HiddenHosts = []string{"from-b"}
	if _, err := SaveInventory(p, &b); err != nil {
		t.Fatal(err)
	}
	a.HiddenHosts = []string{"from-a"}
	if _, err := SaveInventory(p, &a); !errors.Is(err, ErrChangedOnDisk) {
		t.Fatalf("stale save: err=%v, want ErrChangedOnDisk", err)
	}
	if got, _, _ := LoadInventory(p); !reflect.DeepEqual(got.HiddenHosts, []string{"from-b"}) {
		t.Fatalf("file holds %v, want [from-b]", got.HiddenHosts)
	}

	// An instance's own earlier save is not a change on disk.
	b.HiddenHosts = append(b.HiddenHosts, "again")
	if _, err := SaveInventory(p, &b); err != nil {
		t.Fatalf("second save by the same instance: %v", err)
	}
}

func TestSaveConfigRefusesFileChangedOnDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	a, _, _ := Load(p)
	b, _, _ := Load(p)
	if _, err := Save(p, &b); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(p, &a); !errors.Is(err, ErrChangedOnDisk) {
		t.Fatalf("stale save: err=%v, want ErrChangedOnDisk", err)
	}
}

func TestDefaultPathsFollowXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if p, err := DefaultPath(); err != nil || p != filepath.Join(xdg, "sshelf", "config.toml") {
		t.Fatalf("DefaultPath=%q err=%v", p, err)
	}
	if _, used, err := LoadInventory(""); err != nil || used != filepath.Join(xdg, "sshelf", "hosts.toml") {
		t.Fatalf("LoadInventory default path=%q err=%v", used, err)
	}
}

func TestDefaultPathFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	if p, err := DefaultPath(); err != nil || p != filepath.Join(home, ".config", "sshelf", "config.toml") {
		t.Fatalf("DefaultPath=%q err=%v", p, err)
	}
}
