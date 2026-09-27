package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultConfigWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 || !cfg.OpenBrowser {
		t.Fatalf("got %+v, want defaults", cfg)
	}

	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatalf("config.json was not created: %v", err)
	}
}

func TestLoadReadsExistingConfig(t *testing.T) {
	dir := t.TempDir()
	const body = `{"port":9090,"export_dir":"/srv/exports","open_browser":false}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9090 || cfg.OpenBrowser || cfg.ExportDir != "/srv/exports" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadExpandsHomeInExportDir(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	want := filepath.Join(home, "Desktop", "ID Cards")
	if cfg.ExportDir != want {
		t.Fatalf("got ExportDir=%q, want %q", cfg.ExportDir, want)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("write config.json: %v", err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestEnsureDataDirCreatesLayout(t *testing.T) {
	appDir := t.TempDir()
	d, err := EnsureDataDir(appDir)
	if err != nil {
		t.Fatalf("EnsureDataDir: %v", err)
	}
	for _, dir := range []string{d.Root, d.Logos, d.Photos} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}
