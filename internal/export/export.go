// Package export saves browser-rendered card PDFs into month folders under
// the configured export directory and records each export on the
// matching employee.
package export

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"idcardstudio/internal/employees"
	"idcardstudio/internal/storage"
)

// pdfMagic is the PDF file signature: a minimal sanity check, since the
// destination path is server-controlled and not derived from the upload.
var pdfMagic = []byte("%PDF")

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// SanitizeFilenamePart replaces every character outside [A-Za-z0-9_-] with
// "_", per the card filename spec.
func SanitizeFilenamePart(s string) string {
	return unsafeFilenameChars.ReplaceAllString(s, "_")
}

// Filename returns the export filename for an employee: "<id>_<Name>.pdf",
// with Name sanitized.
func Filename(employeeID, name string) string {
	return fmt.Sprintf("%s_%s.pdf", employeeID, SanitizeFilenamePart(name))
}

// MonthFolder formats joinDate ("YYYY-MM-DD") as the fixed-English "Jan
// 2006" folder name used under the export directory, so the folder layout
// is identical on every computer regardless of locale.
func MonthFolder(joinDate string) (string, error) {
	t, err := time.Parse("2006-01-02", joinDate)
	if err != nil {
		return "", fmt.Errorf("join_date %q is not in YYYY-MM-DD format", joinDate)
	}
	return t.Format("Jan 2006"), nil
}

// Manager saves rendered PDFs and records the export on the corresponding
// employee.
type Manager struct {
	Employees *employees.Manager
	ExportDir string
}

// New returns a Manager that saves PDFs under exportDir.
func New(employeesManager *employees.Manager, exportDir string) *Manager {
	return &Manager{Employees: employeesManager, ExportDir: exportDir}
}

// Result describes one saved export.
type Result struct {
	EmployeeID string
	Path       string // absolute path the PDF was saved to
}

// Save validates employeeID, saves pdfData under
// <ExportDir>/<Mon YYYY>/<id>_<Name>.pdf (the month of the employee's
// join_date), and records exported_at/exported_by on the employee.
func (m *Manager) Save(employeeID string, pdfData []byte, exportedBy string) (Result, error) {
	if err := employees.ValidateEmployeeID(employeeID); err != nil {
		return Result{}, err
	}
	if len(pdfData) < len(pdfMagic) || string(pdfData[:len(pdfMagic)]) != string(pdfMagic) {
		return Result{}, errors.New("uploaded file is not a PDF")
	}

	e, found, err := m.Employees.Store.Find(employeeID)
	if err != nil {
		return Result{}, err
	}
	if !found {
		return Result{}, fmt.Errorf("no employee %q", employeeID)
	}

	monthFolder, err := MonthFolder(e.JoinDate)
	if err != nil {
		return Result{}, err
	}
	path := filepath.Join(m.ExportDir, monthFolder, Filename(e.EmployeeID, e.Name))

	if err := storage.WriteFileAtomic(path, pdfData); err != nil {
		return Result{}, fmt.Errorf("save PDF: %w", err)
	}

	now := time.Now()
	if _, err := m.Employees.Store.Update(employeeID, func(cur employees.Employee) (employees.Employee, error) {
		cur.ExportedAt = &now
		cur.ExportedBy = exportedBy
		return cur, nil
	}); err != nil {
		return Result{}, fmt.Errorf("PDF saved but employee record could not be updated: %w", err)
	}

	return Result{EmployeeID: employeeID, Path: path}, nil
}
