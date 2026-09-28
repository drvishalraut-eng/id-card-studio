package backup

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func newTestPaths(t *testing.T) Paths {
	t.Helper()
	dir := t.TempDir()
	return Paths{
		Root:   dir,
		Logos:  filepath.Join(dir, "logos"),
		Photos: filepath.Join(dir, "photos"),
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func seedData(t *testing.T, p Paths) {
	t.Helper()
	writeFile(t, filepath.Join(p.Root, "employees.json"), `[{"employee_id":"E001"}]`)
	writeFile(t, filepath.Join(p.Root, "clients.json"), `[{"id":"acme"}]`)
	writeFile(t, filepath.Join(p.Root, "users.json"), `[{"username":"ada"}]`)
	writeFile(t, filepath.Join(p.Root, "activity.jsonl"), `{"action":"login"}`+"\n")
	writeFile(t, filepath.Join(p.Logos, "acme.svg"), `<svg></svg>`)
	writeFile(t, filepath.Join(p.Photos, "E001.jpg"), "fake-jpeg-bytes")
}

func TestWriteIncludesEveryDataFile(t *testing.T) {
	p := newTestPaths(t)
	seedData(t, p)

	var buf bytes.Buffer
	if err := Write(&buf, p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	want := map[string]string{
		"employees.json":  `[{"employee_id":"E001"}]`,
		"clients.json":    `[{"id":"acme"}]`,
		"users.json":      `[{"username":"ada"}]`,
		"activity.jsonl":  `{"action":"login"}` + "\n",
		"logos/acme.svg":  `<svg></svg>`,
		"photos/E001.jpg": "fake-jpeg-bytes",
	}
	got := map[string]string{}
	for _, f := range zr.File {
		data, err := readZipFile(f)
		if err != nil {
			t.Fatalf("readZipFile(%s): %v", f.Name, err)
		}
		got[f.Name] = string(data)
	}
	for name, contents := range want {
		if got[name] != contents {
			t.Fatalf("zip entry %q = %q, want %q", name, got[name], contents)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("got %d zip entries %v, want exactly %v", len(got), got, want)
	}
}

func TestWriteSkipsMissingFilesAndFoldersOnFreshInstall(t *testing.T) {
	p := newTestPaths(t) // nothing seeded — a genuinely fresh install

	var buf bytes.Buffer
	if err := Write(&buf, p); err != nil {
		t.Fatalf("Write: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if len(zr.File) != 0 {
		t.Fatalf("expected an empty zip for a fresh install with no data, got %d entries", len(zr.File))
	}
}

func buildZip(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, contents := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		if _, err := f.Write([]byte(contents)); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zw.Close: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	return zr
}

func TestImportAppliesAValidBackup(t *testing.T) {
	p := newTestPaths(t)
	zr := buildZip(t, map[string]string{
		"employees.json":  `[{"employee_id":"E002"}]`,
		"clients.json":    `[{"id":"beta"}]`,
		"users.json":      `[{"username":"bea"}]`,
		"activity.jsonl":  `{"action":"restored"}` + "\n",
		"logos/beta.svg":  `<svg>beta</svg>`,
		"photos/E002.jpg": "beta-jpeg",
	})

	if err := Import(zr, p); err != nil {
		t.Fatalf("Import: %v", err)
	}

	assertFileContent(t, filepath.Join(p.Root, "employees.json"), `[{"employee_id":"E002"}]`)
	assertFileContent(t, filepath.Join(p.Root, "clients.json"), `[{"id":"beta"}]`)
	assertFileContent(t, filepath.Join(p.Root, "users.json"), `[{"username":"bea"}]`)
	assertFileContent(t, filepath.Join(p.Root, "activity.jsonl"), `{"action":"restored"}`+"\n")
	assertFileContent(t, filepath.Join(p.Logos, "beta.svg"), `<svg>beta</svg>`)
	assertFileContent(t, filepath.Join(p.Photos, "E002.jpg"), "beta-jpeg")
}

func TestImportReplacesRatherThanMerges(t *testing.T) {
	p := newTestPaths(t)
	seedData(t, p) // has logos/acme.svg and photos/E001.jpg

	zr := buildZip(t, map[string]string{
		"employees.json": `[]`,
		"clients.json":   `[]`,
		"users.json":     `[]`,
		"logos/new.svg":  `<svg>new</svg>`,
	})
	if err := Import(zr, p); err != nil {
		t.Fatalf("Import: %v", err)
	}

	if _, err := os.Stat(filepath.Join(p.Logos, "acme.svg")); !os.IsNotExist(err) {
		t.Fatal("expected the old logo to be removed, not merged with the new backup")
	}
	if _, err := os.Stat(filepath.Join(p.Photos, "E001.jpg")); !os.IsNotExist(err) {
		t.Fatal("expected the old photo to be removed since the backup had no photos")
	}
	if _, err := os.Stat(filepath.Join(p.Root, "activity.jsonl")); !os.IsNotExist(err) {
		t.Fatal("expected activity.jsonl to be removed since the backup had none")
	}
	assertFileContent(t, filepath.Join(p.Logos, "new.svg"), `<svg>new</svg>`)
}

func TestImportRejectsAZipMissingARequiredFile(t *testing.T) {
	p := newTestPaths(t)
	seedData(t, p)

	zr := buildZip(t, map[string]string{
		"employees.json": `[]`,
		"clients.json":   `[]`,
		// users.json missing
	})
	if err := Import(zr, p); err == nil {
		t.Fatal("expected an error for a backup missing users.json")
	}

	// Nothing should have been touched.
	assertFileContent(t, filepath.Join(p.Root, "employees.json"), `[{"employee_id":"E001"}]`)
}

func TestImportRejectsInvalidJSON(t *testing.T) {
	p := newTestPaths(t)
	seedData(t, p)

	zr := buildZip(t, map[string]string{
		"employees.json": `not json`,
		"clients.json":   `[]`,
		"users.json":     `[]`,
	})
	if err := Import(zr, p); err == nil {
		t.Fatal("expected an error for invalid JSON in employees.json")
	}
	assertFileContent(t, filepath.Join(p.Root, "employees.json"), `[{"employee_id":"E001"}]`)
}

func TestImportIgnoresPathTraversalEntries(t *testing.T) {
	p := newTestPaths(t)
	zr := buildZip(t, map[string]string{
		"employees.json":        `[]`,
		"clients.json":          `[]`,
		"users.json":            `[]`,
		"photos/../evil.jpg":    "evil",
		"photos/sub/nested.jpg": "nested",
	})
	if err := Import(zr, p); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p.Photos), "evil.jpg")); !os.IsNotExist(err) {
		t.Fatal("a path-traversal zip entry must not escape the photos directory")
	}
	entries, err := os.ReadDir(p.Photos)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no photos to be written from unsafe entries, got %v", entries)
	}
}

func TestClearResetsToFreshInstallState(t *testing.T) {
	p := newTestPaths(t)
	seedData(t, p)

	if err := Clear(p); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	assertFileContent(t, filepath.Join(p.Root, "employees.json"), `[]`)
	assertFileContent(t, filepath.Join(p.Root, "clients.json"), `[]`)
	assertFileContent(t, filepath.Join(p.Root, "users.json"), `[]`)
	if _, err := os.Stat(filepath.Join(p.Root, "activity.jsonl")); !os.IsNotExist(err) {
		t.Fatal("expected activity.jsonl to be removed")
	}
	for _, dir := range []string{p.Logos, p.Photos} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("ReadDir(%s): %v", dir, err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected %s to be empty, got %v", dir, entries)
		}
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
