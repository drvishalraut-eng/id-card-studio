package export

import (
	"os"
	"path/filepath"
	"testing"

	"idcardstudio/internal/employees"
)

func TestSanitizeFilenamePart(t *testing.T) {
	got := SanitizeFilenamePart(`Priya Rao (HR)/"quote"`)
	want := `Priya_Rao__HR___quote_`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFilename(t *testing.T) {
	got := Filename("E001", "Priya Rao")
	if got != "E001_Priya_Rao.pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestMonthFolderIsFixedEnglish(t *testing.T) {
	got, err := MonthFolder("2026-09-15")
	if err != nil {
		t.Fatalf("MonthFolder: %v", err)
	}
	if got != "Sep 2026" {
		t.Fatalf("got %q, want Sep 2026", got)
	}
}

func TestMonthFolderRejectsBadDate(t *testing.T) {
	if _, err := MonthFolder("not-a-date"); err == nil {
		t.Fatal("expected an error for an invalid join_date")
	}
}

const validPDF = "%PDF-1.4\n%fake-body-for-tests\n%%EOF"

func newTestSetup(t *testing.T) (*Manager, *employees.Manager) {
	t.Helper()
	dir := t.TempDir()
	empMgr := employees.NewManager(filepath.Join(dir, "employees.json"))
	exportDir := filepath.Join(dir, "exports")
	return New(empMgr, exportDir), empMgr
}

func TestSaveWritesPDFUnderMonthFolderAndRecordsExport(t *testing.T) {
	m, empMgr := newTestSetup(t)
	if _, err := empMgr.Save(employees.Input{
		EmployeeID: "E001", Name: "Priya Rao", Client: "Helios", JoinDate: "2026-09-15",
	}, "ada"); err != nil {
		t.Fatalf("Save employee: %v", err)
	}

	result, err := m.Save("E001", []byte(validPDF), "ada")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantPath := filepath.Join(m.ExportDir, "Sep 2026", "E001_Priya_Rao.pdf")
	if result.Path != wantPath {
		t.Fatalf("got path %q, want %q", result.Path, wantPath)
	}
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != validPDF {
		t.Fatal("saved file contents do not match")
	}

	e, found, err := empMgr.Store.Find("E001")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if !found || e.ExportedAt == nil || e.ExportedBy != "ada" {
		t.Fatalf("got %+v, found=%v", e, found)
	}
}

func TestSaveRejectsUnknownEmployee(t *testing.T) {
	m, _ := newTestSetup(t)
	if _, err := m.Save("nope", []byte(validPDF), "ada"); err == nil {
		t.Fatal("expected an error for an unknown employee")
	}
}

func TestSaveRejectsNonPDF(t *testing.T) {
	m, empMgr := newTestSetup(t)
	if _, err := empMgr.Save(employees.Input{EmployeeID: "E001", Name: "Priya Rao", JoinDate: "2026-09-15"}, "ada"); err != nil {
		t.Fatalf("Save employee: %v", err)
	}
	if _, err := m.Save("E001", []byte("not a pdf"), "ada"); err == nil {
		t.Fatal("expected an error for non-PDF content")
	}
}

func TestSaveRejectsInvalidEmployeeID(t *testing.T) {
	m, _ := newTestSetup(t)
	if _, err := m.Save("bad id!", []byte(validPDF), "ada"); err == nil {
		t.Fatal("expected an error for an invalid employee_id")
	}
}
