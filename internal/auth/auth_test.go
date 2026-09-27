package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	return NewManager(filepath.Join(t.TempDir(), "users.json"))
}

func TestSetupCreatesFirstAdmin(t *testing.T) {
	m := newTestManager(t)

	needs, err := m.NeedsSetup()
	if err != nil {
		t.Fatalf("NeedsSetup: %v", err)
	}
	if !needs {
		t.Fatal("expected NeedsSetup to be true before any user exists")
	}

	u, err := m.Setup("Ada Admin", "ada", "1234")
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if u.Role != RoleAdmin {
		t.Fatalf("got role %q, want admin", u.Role)
	}

	needs, err = m.NeedsSetup()
	if err != nil {
		t.Fatalf("NeedsSetup: %v", err)
	}
	if needs {
		t.Fatal("expected NeedsSetup to be false once an admin exists")
	}
}

func TestSetupRejectsSecondCall(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.Setup("Bob", "bob", "5678"); !errors.Is(err, ErrAlreadySetUp) {
		t.Fatalf("got err=%v, want ErrAlreadySetUp", err)
	}
}

func TestSetupRejectsBadPIN(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "12"); err == nil {
		t.Fatal("expected an error for a too-short PIN")
	}
}

func TestLoginSucceedsWithCorrectPIN(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	token, u, err := m.Login("ada", "1234")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty session token")
	}
	if u.LastLogin == nil {
		t.Fatal("expected LastLogin to be set")
	}

	username, ok := m.Sessions.Username(token)
	if !ok || username != "ada" {
		t.Fatalf("session lookup got (%q, %v)", username, ok)
	}
}

func TestLoginFailsWithWrongPIN(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, _, err := m.Login("ada", "0000"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got err=%v, want ErrInvalidCredentials", err)
	}
}

func TestLoginFailsForUnknownUser(t *testing.T) {
	m := newTestManager(t)
	if _, _, err := m.Login("nobody", "1234"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got err=%v, want ErrInvalidCredentials", err)
	}
}

func TestLoginLocksAfterFiveFailedAttempts(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	for i := 0; i < MaxFailedAttempts-1; i++ {
		if _, _, err := m.Login("ada", "0000"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: got err=%v, want ErrInvalidCredentials", i, err)
		}
	}
	// The 5th wrong attempt should trip the lock.
	if _, _, err := m.Login("ada", "0000"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("got err=%v, want ErrAccountLocked", err)
	}
	// Even the correct PIN is rejected while locked.
	if _, _, err := m.Login("ada", "1234"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("got err=%v, want ErrAccountLocked while locked", err)
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := m.Users.Update("ada", func(u User) (User, error) {
		u.Disabled = true
		return u, nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if _, _, err := m.Login("ada", "1234"); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("got err=%v, want ErrAccountDisabled", err)
	}
}

func TestLogoutEndsSession(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	token, _, err := m.Login("ada", "1234")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	m.Logout(token)
	if _, ok := m.Sessions.Username(token); ok {
		t.Fatal("expected session to be gone after Logout")
	}
}

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	m := newTestManager(t)
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run without a valid session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestRequireAuthAllowsValidSession(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	token, _, err := m.Login("ada", "1234")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	var ranAsUser string
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("expected a user in the request context")
		}
		ranAsUser = u.Username
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ranAsUser != "ada" {
		t.Fatalf("handler ran as %q, want ada", ranAsUser)
	}
}

func TestRequireRoleRejectsWrongRole(t *testing.T) {
	m := newTestManager(t)
	if _, err := m.Setup("Ada Admin", "ada", "1234"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := m.Users.Create(User{Username: "opal", Name: "Opal Operator", Role: RoleOperator, PINHash: "x", Salt: "y"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	token, err := m.Sessions.Create("opal")
	if err != nil {
		t.Fatalf("Sessions.Create: %v", err)
	}

	handler := m.RequireRole(RoleAdmin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run for the wrong role")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403", rec.Code)
	}
}

func TestRequireXRequestedWithRejectsMissingHeader(t *testing.T) {
	handler := RequireXRequestedWith(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not run without the header")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestRequireXRequestedWithAllowsCorrectHeader(t *testing.T) {
	ran := false
	handler := RequireXRequestedWith(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.Header.Set(RequestedWithHeader, RequestedWithValue)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !ran {
		t.Fatal("expected the handler to run with the correct header")
	}
}
