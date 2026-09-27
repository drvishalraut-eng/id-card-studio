// Package storage provides mutex-protected, atomically-written JSON files.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store guards a single JSON file with a mutex so concurrent callers never
// interleave reads and writes, and every write lands via a temp file plus
// rename so a crash mid-write can never leave a truncated file on disk.
type Store[T any] struct {
	path string
	mu   sync.Mutex
}

// New returns a Store backed by the JSON file at path. The file and its
// parent directory are created on first write; they need not exist yet.
func New[T any](path string) *Store[T] {
	return &Store[T]{path: path}
}

// Load reads and decodes the JSON file. A missing file is not an error: the
// zero value of T is returned so first-run callers can proceed normally.
func (s *Store[T]) Load() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Save atomically writes v as the file's new contents.
func (s *Store[T]) Save(v T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(v)
}

// Update loads the current value, applies fn to produce the next value, and
// atomically saves it — the whole read-modify-write cycle runs under one
// lock, so two concurrent updates can never race and drop one's change.
func (s *Store[T]) Update(fn func(T) (T, error)) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.load()
	if err != nil {
		var zero T
		return zero, err
	}
	next, err := fn(v)
	if err != nil {
		return next, err
	}
	if err := s.save(next); err != nil {
		return next, err
	}
	return next, nil
}

func (s *Store[T]) load() (T, error) {
	var v T
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return v, nil
	}
	if err != nil {
		return v, fmt.Errorf("read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return v, nil
}

func (s *Store[T]) save(v T) error {
	return WriteAtomic(s.path, v)
}

// WriteAtomic marshals v as indented JSON and writes it to path atomically:
// a temp file in the same directory is written and fsynced, then renamed
// over the target, so readers never observe a partial file.
func WriteAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}
	return nil
}
