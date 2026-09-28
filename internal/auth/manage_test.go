package auth

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAddUserCreatesOperator(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	u, err := m.AddUser("Opal Operator", "opal", RoleOperator, "2222")
	if err != nil {
		t.Fatalf("AddUser: %v", err)
	}
	if u.Role != RoleOperator {
		t.Fatalf("got role %q, want operator", u.Role)
	}

	if _, _, err := m.Login("opal", "2222"); err != nil {
		t.Fatalf("new operator should be able to log in: %v", err)
	}
}

func TestAddUserRejectsInvalidRole(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.AddUser("Bob", "bob", Role("superuser"), "1234"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("got err=%v, want ErrInvalidRole", err)
	}
}

func TestAddUserRejectsDuplicateUsername(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.AddUser("Ada Two", "ADA", RoleOperator, "5678"); err == nil {
		t.Fatal("expected an error for a case-insensitively duplicate username")
	}
}

func TestResetPINAllowsLoginWithNewPINOnly(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if _, err := m.ResetPIN("ada", "9999"); err != nil {
		t.Fatalf("ResetPIN: %v", err)
	}

	if _, _, err := m.Login("ada", "1234"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old PIN should be rejected, got err=%v", err)
	}
	if _, _, err := m.Login("ada", "9999"); err != nil {
		t.Fatalf("new PIN should work: %v", err)
	}
}

func TestResetPINClearsLockout(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	for i := 0; i < MaxFailedAttempts; i++ {
		m.Login("ada", "0000")
	}
	if _, _, err := m.Login("ada", "1234"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected account to be locked, got err=%v", err)
	}

	if _, err := m.ResetPIN("ada", "4321"); err != nil {
		t.Fatalf("ResetPIN: %v", err)
	}
	if _, _, err := m.Login("ada", "4321"); err != nil {
		t.Fatalf("expected login to succeed after PIN reset clears lockout: %v", err)
	}
}

func TestSetDisabledBlocksLastActiveAdmin(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if _, err := m.SetDisabled("ada", true); !errors.Is(err, ErrLastActiveAdmin) {
		t.Fatalf("got err=%v, want ErrLastActiveAdmin", err)
	}
}

func TestSetDisabledAllowsDisablingWhenAnotherAdminIsActive(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.AddUser("Bea Admin", "bea", RoleAdmin, "5678"); err != nil {
		t.Fatalf("AddUser: %v", err)
	}

	if _, err := m.SetDisabled("ada", true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}
	if _, _, err := m.Login("ada", "1234"); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("got err=%v, want ErrAccountDisabled", err)
	}
}

func TestSetDisabledAllowsDisablingOperator(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.AddUser("Opal Operator", "opal", RoleOperator, "2222"); err != nil {
		t.Fatalf("AddUser: %v", err)
	}

	if _, err := m.SetDisabled("opal", true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}
	if _, _, err := m.Login("opal", "2222"); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("got err=%v, want ErrAccountDisabled", err)
	}
}

func TestSetDisabledCanReenable(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.AddUser("Opal Operator", "opal", RoleOperator, "2222"); err != nil {
		t.Fatalf("AddUser: %v", err)
	}
	if _, err := m.SetDisabled("opal", true); err != nil {
		t.Fatalf("SetDisabled(true): %v", err)
	}
	if _, err := m.SetDisabled("opal", false); err != nil {
		t.Fatalf("SetDisabled(false): %v", err)
	}
	if _, _, err := m.Login("opal", "2222"); err != nil {
		t.Fatalf("expected re-enabled operator to log in: %v", err)
	}
}

// A nil slice marshals to JSON null, not []; a fresh install with no
// users.json yet must still return a real (non-nil) empty slice. In
// practice /api/users can't be reached before Setup creates the first
// admin, but List() itself should still hold this invariant.
func TestListOnMissingFileReturnsEmptySliceNotNil(t *testing.T) {
	m := newTestManager(t)
	list, err := m.Users.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Fatal("List() returned nil; it must return a non-nil empty slice so it marshals to JSON [] not null")
	}
	data, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("got JSON %s, want []", data)
	}
}
