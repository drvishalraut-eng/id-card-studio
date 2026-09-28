package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"idcardstudio/internal/activity"
	"idcardstudio/internal/auth"
	"idcardstudio/internal/clients"
	"idcardstudio/internal/employees"
	"idcardstudio/internal/export"
	"idcardstudio/internal/presence"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	exportDir := filepath.Join(dir, "exports")
	employeesManager := employees.NewManager(filepath.Join(dir, "employees.json"))
	return New(
		auth.NewManager(filepath.Join(dir, "users.json")),
		activity.New(filepath.Join(dir, "activity.jsonl")),
		presence.New(),
		clients.NewManager(filepath.Join(dir, "clients.json"), filepath.Join(dir, "logos")),
		employeesManager,
		export.New(employeesManager, exportDir),
		Info{
			StartedAt: time.Now(), Port: 8080, DataDir: dir,
			ExportDir: exportDir, PhotosDir: filepath.Join(dir, "photos"),
		},
	)
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSetupStatusReportsNeeded(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodGet, "/setup", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var got map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got["needed"] {
		t.Fatal("expected needed=true before any user exists")
	}
}

func TestSetupThenLoginThenMeThenLogout(t *testing.T) {
	h := newTestServer(t).Handler()

	setupRec := doJSON(t, h, http.MethodPost, "/setup", map[string]string{
		"name": "Ada Admin", "username": "ada", "pin": "1234",
	})
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup: got status %d, body %s", setupRec.Code, setupRec.Body)
	}
	sessionCookie := findCookie(t, setupRec, auth.CookieName)

	meRec := doJSON(t, h, http.MethodGet, "/me", nil, sessionCookie)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me: got status %d, body %s", meRec.Code, meRec.Body)
	}
	var me userResponse
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if me.Username != "ada" || me.Role != auth.RoleAdmin {
		t.Fatalf("got %+v", me)
	}

	logoutRec := doJSON(t, h, http.MethodPost, "/logout", nil, sessionCookie)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout: got status %d", logoutRec.Code)
	}

	meAfterRec := doJSON(t, h, http.MethodGet, "/me", nil, sessionCookie)
	if meAfterRec.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout: got status %d, want 401", meAfterRec.Code)
	}
}

func TestSetupRejectsWhenAlreadyDone(t *testing.T) {
	h := newTestServer(t).Handler()
	doJSON(t, h, http.MethodPost, "/setup", map[string]string{"name": "Ada", "username": "ada", "pin": "1234"})

	rec := doJSON(t, h, http.MethodPost, "/setup", map[string]string{"name": "Bob", "username": "bob", "pin": "5678"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("got status %d, want 409", rec.Code)
	}
}

func TestLoginRejectsWrongPIN(t *testing.T) {
	h := newTestServer(t).Handler()
	doJSON(t, h, http.MethodPost, "/setup", map[string]string{"name": "Ada", "username": "ada", "pin": "1234"})

	rec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "ada", "pin": "0000"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestLoginLocksAccountAfterFiveFailures(t *testing.T) {
	h := newTestServer(t).Handler()
	doJSON(t, h, http.MethodPost, "/setup", map[string]string{"name": "Ada", "username": "ada", "pin": "1234"})

	for i := 0; i < auth.MaxFailedAttempts-1; i++ {
		doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "ada", "pin": "0000"})
	}
	rec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "ada", "pin": "0000"})
	if rec.Code != http.StatusLocked {
		t.Fatalf("got status %d, want 423", rec.Code)
	}
}

func TestRoutesRejectMissingRequestedWithHeader(t *testing.T) {
	h := newTestServer(t).Handler()
	req := httptest.NewRequest(http.MethodGet, "/setup", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestRoutesRejectOversizedBody(t *testing.T) {
	h := newTestServer(t).Handler()
	oversized := strings.Repeat("a", MaxRequestBody+1)
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(oversized))
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusCreated {
		t.Fatalf("expected an oversized body to be rejected, got status %d", rec.Code)
	}
}

func findCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no %q cookie in response", name)
	return nil
}
