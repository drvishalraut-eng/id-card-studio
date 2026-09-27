package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"idcardstudio/internal/auth"
)

func setupAdmin(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/setup", map[string]string{
		"name": "Ada Admin", "username": "ada", "pin": "1234",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: got status %d, body %s", rec.Code, rec.Body)
	}
	return findCookie(t, rec, auth.CookieName)
}

func TestListUsersRequiresAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodGet, "/users", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401 without a session", rec.Code)
	}
}

func TestListUsersReturnsAllUsers(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add user: got status %d, body %s", rec.Code, rec.Body)
	}

	listRec := doJSON(t, h, http.MethodGet, "/users", nil, admin)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: got status %d, body %s", listRec.Code, listRec.Body)
	}
	var items []userListItem
	if err := json.Unmarshal(listRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d users, want 2", len(items))
	}
}

func TestAddUserRejectsMismatchedConfirmPIN(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "3333",
	}, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestAddUserRejectsNonAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)

	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	opalCookie := findCookie(t, loginRec, auth.CookieName)

	rec := doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Bob", "username": "bob", "role": "operator", "pin": "1111", "confirm_pin": "1111",
	}, opalCookie)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403 for a non-admin caller", rec.Code)
	}
}

func TestResetPINChangesCredentials(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)

	rec := doJSON(t, h, http.MethodPost, "/users/opal/reset-pin", map[string]string{"pin": "9999", "confirm_pin": "9999"}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset-pin: got status %d, body %s", rec.Code, rec.Body)
	}

	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "9999"})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login with new PIN: got status %d", loginRec.Code)
	}
}

func TestDisableThenEnableUser(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)

	disableRec := doJSON(t, h, http.MethodPost, "/users/opal/disable", nil, admin)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("disable: got status %d, body %s", disableRec.Code, disableRec.Body)
	}
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	if loginRec.Code != http.StatusForbidden {
		t.Fatalf("login while disabled: got status %d, want 403", loginRec.Code)
	}

	enableRec := doJSON(t, h, http.MethodPost, "/users/opal/enable", nil, admin)
	if enableRec.Code != http.StatusOK {
		t.Fatalf("enable: got status %d, body %s", enableRec.Code, enableRec.Body)
	}
	loginRec2 := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	if loginRec2.Code != http.StatusOK {
		t.Fatalf("login after re-enable: got status %d", loginRec2.Code)
	}
}

func TestDisableLastAdminIsRejected(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/users/ada/disable", nil, admin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got status %d, want 409 for disabling the last active admin", rec.Code)
	}
}
