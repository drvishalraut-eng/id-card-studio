// Package config loads the portable config.json and lays out the data
// folder next to the running executable, so the app can run from any
// folder or USB drive without installation.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"idcardstudio/internal/storage"
)

// Config is the contents of config.json.
type Config struct {
	Port        int    `json:"port"`
	ExportDir   string `json:"export_dir"`
	OpenBrowser bool   `json:"open_browser"`
}

// Default returns the config used when no config.json exists yet.
func Default() Config {
	return Config{Port: 8080, ExportDir: "~/Desktop/ID Cards", OpenBrowser: true}
}

// Load reads config.json from dir, creating it with default values if it
// does not exist yet. ExportDir has a leading "~" expanded to the user's
// home folder.
func Load(dir string) (Config, error) {
	path := filepath.Join(dir, "config.json")
	cfg := Default()

	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		if err := storage.WriteAtomic(path, cfg); err != nil {
			return Config{}, fmt.Errorf("create default config.json: %w", err)
		}
	case err != nil:
		return Config{}, fmt.Errorf("read config.json: %w", err)
	default:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("config.json is not valid JSON: %w", err)
		}
	}

	expanded, err := expandHome(cfg.ExportDir)
	if err != nil {
		return Config{}, err
	}
	cfg.ExportDir = expanded
	return cfg, nil
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home folder: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}

// AppDir returns the directory containing the running executable, resolving
// symlinks so the app finds its neighboring data folder even when launched
// via a shortcut.
func AppDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate running executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Dir(resolved), nil
}

// DataDir is the layout of the portable data/ folder next to the executable.
type DataDir struct {
	Root   string
	Logos  string
	Photos string
}

// EnsureDataDir creates the data/, data/logos/ and data/photos/ folders
// under appDir if they do not already exist.
func EnsureDataDir(appDir string) (DataDir, error) {
	d := DataDir{
		Root:   filepath.Join(appDir, "data"),
		Logos:  filepath.Join(appDir, "data", "logos"),
		Photos: filepath.Join(appDir, "data", "photos"),
	}
	for _, dir := range []string{d.Root, d.Logos, d.Photos} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return DataDir{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return d, nil
}
