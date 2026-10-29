package config

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// configFile returns name inside $XDG_CONFIG_HOME/sshelf, or ~/.config/sshelf.
func configFile(name string) (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "sshelf", name), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("home directory not found")
	}
	return filepath.Join(home, ".config", "sshelf", name), nil
}

func DefaultPath() (string, error) { return configFile("config.toml") }

// defaultHostsPath returns the default path for the hosts inventory file.
func defaultHostsPath() (string, error) { return configFile("hosts.toml") }

// HostsPathFromConfigPath derives the hosts.toml path from a config.toml path
// by replacing the filename in the same directory.
func HostsPathFromConfigPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "hosts.toml")
}

// ErrChangedOnDisk means another process wrote the file after it was loaded;
// saving would overwrite that change.
var ErrChangedOnDisk = errors.New("changed on disk since sshelf read it; restart sshelf to reload")

// fileSum is the SHA-256 of a file as last read or written; zero means no file.
type fileSum [sha256.Size]byte

// loadTOML decodes path over the defaults in v and records the file's sum.
// A missing file is not an error: v keeps its defaults. kind names the file
// in error messages.
func loadTOML[T any](path string, defaultPath func() (string, error), kind string, v *T, sum *fileSum) (string, error) {
	if path == "" {
		p, err := defaultPath()
		if err != nil {
			return "", err
		}
		path = p
	}

	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return path, err
	}
	if st.IsDir() {
		return path, fmt.Errorf("%s path is a directory: %s", kind, path)
	}

	// #nosec G304 -- path is user-provided by design (CLI flag / default config dir).
	data, err := os.ReadFile(path)
	if err != nil {
		return path, err
	}
	if _, err := toml.Decode(string(data), v); err != nil {
		return path, err
	}
	*sum = sha256.Sum256(data)
	return path, nil
}

// saveTOML writes v to path atomically, with 0600 permissions. It refuses
// with ErrChangedOnDisk when the file no longer matches sum, and updates sum
// after the write.
func saveTOML[T any](path string, defaultPath func() (string, error), tmpPattern string, v T, sum *fileSum) (string, error) {
	if path == "" {
		p, err := defaultPath()
		if err != nil {
			return "", err
		}
		path = p
	}
	path = filepath.Clean(path)

	// Write through a symlink (e.g. into a dotfiles repo) instead of replacing it.
	target := path
	if p, err := filepath.EvalSymlinks(path); err == nil {
		target = p
	}
	if err := checkUnchanged(target, *sum); err != nil {
		return path, fmt.Errorf("%s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return path, err
	}
	if err := writeAtomic(target, tmpPattern, buf.Bytes()); err != nil {
		return path, err
	}
	*sum = sha256.Sum256(buf.Bytes())
	return path, nil
}

// writeAtomic replaces target with data through a temp file in the same dir,
// so a crash leaves either the old or the new file, never half of one.
func writeAtomic(target, tmpPattern string, data []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, tmpPattern)
	if err != nil {
		return err
	}
	tmpPath := filepath.Clean(tmp.Name())
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	// CreateTemp makes the file 0600; the rename keeps that mode.
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// #nosec G703 -- target is the cleaned config path with symlinks resolved; taint propagation is a gosec limitation.
	return os.Rename(tmpPath, target)
}

// checkUnchanged returns ErrChangedOnDisk when the file at path differs from sum.
func checkUnchanged(path string, sum fileSum) error {
	// #nosec G304 -- path is the config file being saved.
	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		return nil // nothing on disk to overwrite
	case err != nil:
		return err
	case sha256.Sum256(data) != sum:
		return ErrChangedOnDisk
	}
	return nil
}

// Load reads config.toml. If path is empty, DefaultPath is used.
// A missing file is not an error — DefaultConfig is returned.
func Load(path string) (Config, string, error) {
	cfg := DefaultConfig()
	used, err := loadTOML(path, DefaultPath, "config", &cfg, &cfg.sum)
	if err != nil {
		return DefaultConfig(), used, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, used, nil
}

// Save atomically writes the config to path. It fails with ErrChangedOnDisk
// if the file changed since cfg was loaded or saved.
func Save(path string, cfg *Config) (string, error) {
	cfg.Version = 1
	return saveTOML(path, DefaultPath, ".config.toml.*", *cfg, &cfg.sum)
}

// LoadInventory loads the hosts/groups inventory from path.
// If path is empty, defaultHostsPath is used.
// A missing file is not an error — DefaultInventory is returned.
func LoadInventory(path string) (Inventory, string, error) {
	inv := DefaultInventory()
	used, err := loadTOML(path, defaultHostsPath, "hosts", &inv, &inv.sum)
	if err != nil {
		return DefaultInventory(), used, err
	}
	if inv.Version == 0 {
		inv.Version = 1
	}
	for _, g := range inv.Groups {
		if err := ValidateGroupName(g.Name); err != nil {
			return DefaultInventory(), used, fmt.Errorf("hosts: group %q: %w", g.Name, err)
		}
	}
	return inv, used, nil
}

// SaveInventory atomically writes the inventory to path. It fails with
// ErrChangedOnDisk if the file changed since inv was loaded or saved.
// If path is empty, defaultHostsPath is used.
func SaveInventory(path string, inv *Inventory) (string, error) {
	inv.Version = 1
	return saveTOML(path, defaultHostsPath, ".hosts.toml.*", *inv, &inv.sum)
}
