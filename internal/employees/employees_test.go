package employees

import (
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(filepath.Join(t.TempDir(), "employees.json"))
}

func TestSaveCreatesNewEmployeeWithDefaultCrop(t *testing.T) {
	m := newTestManager(t)
	e, err := m.Save(Input{
		EmployeeID: "E001", Name: "Priya Rao", Role: "Forklift Operator",
		Client: "Helios", JoinDate: "2026-09-15", PhotoNote: "glasses",
	}, "ada")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if e.JoinDate != "2026-09-15" {
		t.Fatalf("got JoinDate=%q", e.JoinDate)
	}
	if e.Crop != DefaultCrop() {
		t.Fatalf("got Crop=%+v, want default", e.Crop)
	}
	if e.UpdatedBy != "ada" {
		t.Fatalf("got UpdatedBy=%q", e.UpdatedBy)
	}
}

func TestSaveRejectsInvalidEmployeeID(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "bad id!", Name: "X", JoinDate: "2026-01-01"}, "ada"); err == nil {
		t.Fatal("expected an error for an invalid employee_id")
	}
}

func TestSaveRejectsMissingName(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "E001", JoinDate: "2026-01-01"}, "ada"); err == nil {
		t.Fatal("expected an error for a missing name")
	}
}

func TestSaveRejectsBadJoinDate(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "E001", Name: "X", JoinDate: "not-a-date"}, "ada"); err == nil {
		t.Fatal("expected an error for an unparseable join date")
	}
}

func TestSaveUpsertsPreservingPhotoAndCrop(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "E001", Name: "Priya Rao", JoinDate: "2026-01-01"}, "ada"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := m.Store.Update("E001", func(e Employee) (Employee, error) {
		e.Photo = "E001.jpg"
		e.Crop = Crop{Zoom: 1.5, X: 40, Y: 60}
		return e, nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := m.Save(Input{EmployeeID: "E001", Name: "Priya R. Rao", JoinDate: "2026-01-02"}, "bob")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if updated.Name != "Priya R. Rao" || updated.JoinDate != "2026-01-02" {
		t.Fatalf("got %+v", updated)
	}
	if updated.Photo != "E001.jpg" || updated.Crop.Zoom != 1.5 {
		t.Fatalf("expected photo/crop to survive an edit-form re-save, got %+v", updated)
	}
	if updated.UpdatedBy != "bob" {
		t.Fatalf("got UpdatedBy=%q, want bob", updated.UpdatedBy)
	}
}

func TestMapClientCorrectsClientCode(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Save(Input{EmployeeID: "E001", Name: "Priya Rao", Client: "helioss", JoinDate: "2026-01-01"}, "ada"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	updated, err := m.MapClient("E001", "Helios", "ada")
	if err != nil {
		t.Fatalf("MapClient: %v", err)
	}
	if updated.ClientCode != "Helios" {
		t.Fatalf("got ClientCode=%q", updated.ClientCode)
	}
}

func TestMapClientRejectsUnknownEmployee(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.MapClient("nope", "Helios", "ada"); err == nil {
		t.Fatal("expected an error for an unknown employee_id")
	}
}
