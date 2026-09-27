// Package employees manages the employee roster: upsert-by-ID, the
// employee_id format shared with export filenames, and each employee's
// photo crop settings.
package employees

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"idcardstudio/internal/storage"
)

// IDPattern is the employee_id format required everywhere it's used:
// uploads, exports and path construction.
var IDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// ValidateEmployeeID reports whether id matches IDPattern, with a
// human-readable error naming the problem when it does not.
func ValidateEmployeeID(id string) error {
	if !IDPattern.MatchString(id) {
		return errors.New("employee ID must be 1-32 characters: letters, numbers, underscore or hyphen only")
	}
	return nil
}

// Crop is the photo crop applied on the card front.
type Crop struct {
	Zoom float64 `json:"zoom"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// DefaultCrop is used for a newly created employee or a freshly uploaded
// photo: no zoom, centered.
func DefaultCrop() Crop {
	return Crop{Zoom: 1, X: 50, Y: 50}
}

// Employee is one row of employees.json.
type Employee struct {
	EmployeeID string     `json:"employee_id"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	ClientCode string     `json:"client_code"`
	JoinDate   string     `json:"join_date"` // YYYY-MM-DD
	PhotoNote  string     `json:"photo_note"`
	Photo      string     `json:"photo"` // filename under data/photos/, empty if none
	Crop       Crop       `json:"crop"`
	UpdatedBy  string     `json:"updated_by"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ExportedAt *time.Time `json:"exported_at,omitempty"`
	ExportedBy string     `json:"exported_by,omitempty"`
}

// Store is the mutex-protected, atomically-written employees.json file.
type Store struct {
	store *storage.Store[[]Employee]
}

// NewStore opens (or prepares to create) the employees.json file at path.
func NewStore(path string) *Store {
	return &Store{store: storage.New[[]Employee](path)}
}

// List returns every employee, in no particular order.
func (s *Store) List() ([]Employee, error) {
	return s.store.Load()
}

// Find returns the employee with the given employee_id, if any.
func (s *Store) Find(id string) (Employee, bool, error) {
	employees, err := s.store.Load()
	if err != nil {
		return Employee{}, false, err
	}
	for _, e := range employees {
		if e.EmployeeID == id {
			return e, true, nil
		}
	}
	return Employee{}, false, nil
}

// Upsert inserts e if its employee_id is new, or replaces the existing
// entry with the same employee_id.
func (s *Store) Upsert(e Employee) error {
	_, err := s.store.Update(func(employees []Employee) ([]Employee, error) {
		for i, existing := range employees {
			if existing.EmployeeID == e.EmployeeID {
				employees[i] = e
				return employees, nil
			}
		}
		return append(employees, e), nil
	})
	return err
}

// Update applies fn to the employee matching id and atomically saves the
// result.
func (s *Store) Update(id string, fn func(Employee) (Employee, error)) (Employee, error) {
	var updated Employee
	_, err := s.store.Update(func(employees []Employee) ([]Employee, error) {
		for i, e := range employees {
			if e.EmployeeID != id {
				continue
			}
			next, err := fn(e)
			if err != nil {
				return employees, err
			}
			employees[i] = next
			updated = next
			return employees, nil
		}
		return employees, fmt.Errorf("no employee %q", id)
	})
	return updated, err
}

// Manager validates employee input before writing it to the Store.
type Manager struct {
	Store *Store
}

// NewManager returns a Manager backed by the given employees.json path.
func NewManager(path string) *Manager {
	return &Manager{Store: NewStore(path)}
}

// Input is the editable fields of an employee, as submitted by the
// add-employee form, an edit, or one row of a bulk import.
type Input struct {
	EmployeeID string
	Name       string
	Role       string
	Client     string
	JoinDate   any // string ("YYYY-MM-DD" or "DD-MM-YYYY") or a float64 Excel serial
	PhotoNote  string
}

// Save validates input and creates or updates the matching employee,
// preserving its photo, crop and export metadata when it already exists.
func (m *Manager) Save(input Input, updatedBy string) (Employee, error) {
	id := strings.TrimSpace(input.EmployeeID)
	if err := ValidateEmployeeID(id); err != nil {
		return Employee{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Employee{}, errors.New("name is required")
	}
	joinDate, err := ParseDateValue(input.JoinDate)
	if err != nil {
		return Employee{}, err
	}

	existing, found, err := m.Store.Find(id)
	if err != nil {
		return Employee{}, err
	}
	e := existing
	if !found {
		e = Employee{EmployeeID: id, Crop: DefaultCrop()}
	}
	e.Name = name
	e.Role = strings.TrimSpace(input.Role)
	e.ClientCode = strings.TrimSpace(input.Client)
	e.JoinDate = joinDate
	e.PhotoNote = strings.TrimSpace(input.PhotoNote)
	e.UpdatedBy = updatedBy
	e.UpdatedAt = time.Now()

	if err := m.Store.Upsert(e); err != nil {
		return Employee{}, err
	}
	return e, nil
}

// MapClient corrects the client_code of an existing employee — the
// "Needs attention" dropdown action.
func (m *Manager) MapClient(id, clientCode, updatedBy string) (Employee, error) {
	clientCode = strings.TrimSpace(clientCode)
	if clientCode == "" {
		return Employee{}, errors.New("client is required")
	}
	return m.Store.Update(id, func(e Employee) (Employee, error) {
		e.ClientCode = clientCode
		e.UpdatedBy = updatedBy
		e.UpdatedAt = time.Now()
		return e, nil
	})
}
