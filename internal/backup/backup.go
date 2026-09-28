// Package backup exports and imports the entire portable data/ folder as a
// single zip file, and can wipe it back to a fresh-install state. These are
// the three whole-system operations the Admin-only Data page offers; none
// of them touch config.json or the export folder, which are host/machine
// settings rather than the app's own data.
package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"idcardstudio/internal/storage"
)

// Paths is the set of data/ locations a backup covers.
type Paths struct {
	Root   string // employees.json, clients.json, users.json, activity.jsonl
	Logos  string // logos/<client_id>.svg
	Photos string // photos/<employee_id>.jpg
}

// jsonFiles are the top-level files every backup round-trips. A zip missing
// any of these is rejected on import — it doesn't look like a real backup.
var jsonFiles = []string{"employees.json", "clients.json", "users.json"}

const activityFile = "activity.jsonl"

// Write streams a zip of every file under paths to w: the three top-level
// JSON files, activity.jsonl, and every file under logos/ and photos/.
func Write(w io.Writer, paths Paths) error {
	zw := zip.NewWriter(w)

	for _, name := range jsonFiles {
		if err := addFile(zw, filepath.Join(paths.Root, name), name); err != nil {
			return err
		}
	}
	if err := addFile(zw, filepath.Join(paths.Root, activityFile), activityFile); err != nil {
		return err
	}
	if err := addDir(zw, paths.Logos, "logos"); err != nil {
		return err
	}
	if err := addDir(zw, paths.Photos, "photos"); err != nil {
		return err
	}
	return zw.Close()
}

// addFile adds src to the zip at zipPath if it exists; a missing file (e.g.
// a fresh install with no activity logged yet) is silently skipped rather
// than failing the whole export.
func addFile(zw *zip.Writer, src, zipPath string) error {
	data, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	f, err := zw.Create(zipPath)
	if err != nil {
		return fmt.Errorf("add %s to zip: %w", zipPath, err)
	}
	_, err = f.Write(data)
	return err
}

func addDir(zw *zip.Writer, dir, zipPrefix string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := addFile(zw, filepath.Join(dir, e.Name()), zipPrefix+"/"+e.Name()); err != nil {
			return err
		}
	}
	return nil
}

// Import replaces everything under paths with zr's contents. The three
// top-level JSON files must be present and must each be valid JSON, or
// Import fails without writing anything; activity.jsonl, logos/*.svg and
// photos/*.jpg are optional. Whatever isn't in the zip is removed from
// paths afterwards, so the result exactly matches the backup rather than
// merging it with whatever was there before.
func Import(zr *zip.Reader, paths Paths) error {
	byName := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		byName[f.Name] = f
	}

	required := make(map[string][]byte, len(jsonFiles))
	for _, name := range jsonFiles {
		f, ok := byName[name]
		if !ok {
			return fmt.Errorf("backup is missing %s — this doesn't look like an ID Card Studio backup", name)
		}
		data, err := readZipFile(f)
		if err != nil {
			return fmt.Errorf("read %s from backup: %w", name, err)
		}
		if !json.Valid(data) {
			return fmt.Errorf("%s in the backup is not valid JSON", name)
		}
		required[name] = data
	}

	// Every required file validated — now actually apply the backup.
	for _, name := range jsonFiles {
		if err := storage.WriteFileAtomic(filepath.Join(paths.Root, name), required[name]); err != nil {
			return err
		}
	}

	activityPath := filepath.Join(paths.Root, activityFile)
	if f, ok := byName[activityFile]; ok {
		data, err := readZipFile(f)
		if err != nil {
			return fmt.Errorf("read %s from backup: %w", activityFile, err)
		}
		if err := storage.WriteFileAtomic(activityPath, data); err != nil {
			return err
		}
	} else if err := os.Remove(activityPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", activityFile, err)
	}

	if err := replaceDir(paths.Logos, byName, "logos/"); err != nil {
		return err
	}
	return replaceDir(paths.Photos, byName, "photos/")
}

// replaceDir empties dir and refills it with every zip entry under prefix,
// ignoring any entry whose remaining name isn't a plain file name — that
// silently rejects directory traversal or nested-path entries in a
// tampered or corrupt zip.
func replaceDir(dir string, byName map[string]*zip.File, prefix string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	for name, f := range byName {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		if rest == "" || strings.ContainsAny(rest, "/\\") {
			continue
		}
		data, err := readZipFile(f)
		if err != nil {
			return fmt.Errorf("read %s from backup: %w", name, err)
		}
		if err := storage.WriteFileAtomic(filepath.Join(dir, rest), data); err != nil {
			return err
		}
	}
	return nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// Clear wipes every file under paths back to a fresh-install state: the
// three JSON files become an empty array, activity.jsonl is removed, and
// every logo and photo is deleted.
func Clear(paths Paths) error {
	for _, name := range jsonFiles {
		if err := storage.WriteFileAtomic(filepath.Join(paths.Root, name), []byte("[]")); err != nil {
			return err
		}
	}
	if err := os.Remove(filepath.Join(paths.Root, activityFile)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", activityFile, err)
	}
	for _, dir := range []string{paths.Logos, paths.Photos} {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("clear %s: %w", dir, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("recreate %s: %w", dir, err)
		}
	}
	return nil
}
