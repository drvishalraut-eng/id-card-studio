package storage

import (
	"path/filepath"
	"sync"
	"testing"
)

type widget struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestLoadMissingFileReturnsZeroValue(t *testing.T) {
	s := New[widget](filepath.Join(t.TempDir(), "missing.json"))
	v, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v != (widget{}) {
		t.Fatalf("expected zero value, got %+v", v)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	s := New[widget](filepath.Join(t.TempDir(), "w.json"))
	want := widget{Name: "gizmo", Count: 3}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSaveCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "w.json")
	s := New[widget](path)
	if err := s.Save(widget{Name: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "x" {
		t.Fatalf("got %+v", got)
	}
}

func TestConcurrentUpdatesDoNotLoseWrites(t *testing.T) {
	s := New[widget](filepath.Join(t.TempDir(), "counter.json"))
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, err := s.Update(func(v widget) (widget, error) {
				v.Count++
				return v, nil
			}); err != nil {
				t.Errorf("Update: %v", err)
			}
		}()
	}
	wg.Wait()

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Count != goroutines {
		t.Fatalf("got Count=%d, want %d", got.Count, goroutines)
	}
}

func TestWriteAtomicOverwritesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.json")
	if err := WriteAtomic(path, widget{Name: "first"}); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	if err := WriteAtomic(path, widget{Name: "second"}); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	s := New[widget](path)
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "second" {
		t.Fatalf("got %+v", got)
	}
}
