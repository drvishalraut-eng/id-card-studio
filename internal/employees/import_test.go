package employees

import "testing"

func TestImportUpsertsValidRowsAndSkipsBadOnes(t *testing.T) {
	m := newTestManager(t)
	rows := []ImportRow{
		{EmployeeID: "E001", Name: "Priya Rao", Client: "Helios", JoinDate: "2026-01-15"},
		{EmployeeID: "", Name: "No ID", JoinDate: "2026-01-15"},
		{EmployeeID: "E002", Name: "", JoinDate: "2026-01-15"},
		{EmployeeID: "E003", Name: "Bad Date", JoinDate: "not a date"},
		{EmployeeID: "E004", Name: "Excel Serial", JoinDate: float64(46280)},
	}

	result, err := m.Import(rows, "ada")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 2 {
		t.Fatalf("got Imported=%d, want 2", result.Imported)
	}
	if len(result.Skipped) != 3 {
		t.Fatalf("got %d skipped rows, want 3: %+v", len(result.Skipped), result.Skipped)
	}
	// Row numbers are 1-based and match input order.
	wantRows := map[int]bool{2: true, 3: true, 4: true}
	for _, s := range result.Skipped {
		if !wantRows[s.Row] {
			t.Errorf("unexpected skipped row number %d (reason %q)", s.Row, s.Reason)
		}
		if s.Reason == "" {
			t.Errorf("row %d has an empty skip reason", s.Row)
		}
	}

	e4, found, err := m.Store.Find("E004")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if !found || e4.JoinDate != "2026-09-15" {
		t.Fatalf("got %+v, found=%v", e4, found)
	}
}

func TestImportUpsertsExistingEmployeeByID(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "E001", Name: "Old Name", JoinDate: "2026-01-01"}, "ada"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	result, err := m.Import([]ImportRow{
		{EmployeeID: "E001", Name: "New Name", JoinDate: "2026-02-01"},
	}, "bob")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("got Imported=%d, want 1", result.Imported)
	}

	list, err := m.Store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "New Name" {
		t.Fatalf("got %+v, want a single updated employee", list)
	}
}

func TestImportAcceptsUnknownClientCodeForLaterMapping(t *testing.T) {
	m := newTestManager(t)
	result, err := m.Import([]ImportRow{
		{EmployeeID: "E001", Name: "Priya Rao", Client: "TotallyUnknownCo", JoinDate: "2026-01-01"},
	}, "ada")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("expected an unknown client to still import (for later mapping), got %+v", result)
	}
}
