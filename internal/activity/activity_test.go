package activity

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestQueryOnMissingFileReturnsEmpty(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	res, err := l.Query(Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != 0 || len(res.Entries) != 0 {
		t.Fatalf("got %+v, want empty result", res)
	}
}

func TestWriteThenQueryReturnsEntry(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	if err := l.Write(Entry{Username: "ada", IP: "10.0.0.5", Action: "sign_in", Target: "ada"}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	res, err := l.Query(Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != 1 || len(res.Entries) != 1 {
		t.Fatalf("got %+v", res)
	}
	if res.Entries[0].Username != "ada" || res.Entries[0].Action != "sign_in" {
		t.Fatalf("got %+v", res.Entries[0])
	}
	if res.Entries[0].TS.IsZero() {
		t.Fatal("expected TS to be auto-filled")
	}
}

func TestQueryOrdersNewestFirst(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l.Write(Entry{TS: base, Action: "first"})
	l.Write(Entry{TS: base.Add(time.Hour), Action: "second"})
	l.Write(Entry{TS: base.Add(2 * time.Hour), Action: "third"})

	res, err := l.Query(Filter{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(res.Entries))
	}
	got := []string{res.Entries[0].Action, res.Entries[1].Action, res.Entries[2].Action}
	want := []string{"third", "second", "first"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got order %v, want %v", got, want)
		}
	}
}

func TestQueryFiltersByUsername(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	l.Write(Entry{Username: "ada", Action: "sign_in"})
	l.Write(Entry{Username: "bob", Action: "sign_in"})

	res, err := l.Query(Filter{Username: "ADA"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != 1 || res.Entries[0].Username != "ada" {
		t.Fatalf("got %+v", res)
	}
}

func TestQueryFiltersByDateRange(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	jan1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	mar1 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	l.Write(Entry{TS: jan1, Action: "jan"})
	l.Write(Entry{TS: feb1, Action: "feb"})
	l.Write(Entry{TS: mar1, Action: "mar"})

	res, err := l.Query(Filter{From: jan1.Add(time.Hour), To: feb1.Add(time.Hour)})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != 1 || res.Entries[0].Action != "feb" {
		t.Fatalf("got %+v", res)
	}
}

func TestQuerySearchMatchesAnyField(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	l.Write(Entry{Username: "ada", Action: "client_added", Target: "Helios Material Handling"})
	l.Write(Entry{Username: "bob", Action: "sign_in", Target: "bob"})

	res, err := l.Query(Filter{Search: "helios"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != 1 || res.Entries[0].Username != "ada" {
		t.Fatalf("got %+v", res)
	}
}

func TestQueryPagination(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		l.Write(Entry{TS: base.Add(time.Duration(i) * time.Minute), Action: "e"})
	}

	page1, err := l.Query(Filter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if page1.Total != 5 || len(page1.Entries) != 2 {
		t.Fatalf("page1: got %+v", page1)
	}

	page3, err := l.Query(Filter{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if page3.Total != 5 || len(page3.Entries) != 1 {
		t.Fatalf("page3: got %+v", page3)
	}
}

func TestConcurrentWritesAllPersist(t *testing.T) {
	l := New(filepath.Join(t.TempDir(), "activity.jsonl"))
	const n = 40
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if err := l.Write(Entry{Action: "concurrent"}); err != nil {
				t.Errorf("Write: %v", err)
			}
		}()
	}
	wg.Wait()

	res, err := l.Query(Filter{PageSize: n})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Total != n {
		t.Fatalf("got Total=%d, want %d", res.Total, n)
	}
}
