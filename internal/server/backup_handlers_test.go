package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"idcardstudio/internal/auth"
)

func setupOperator(t *testing.T, h http.Handler, admin *http.Cookie) *http.Cookie {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/users", map[string]any{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add operator: got status %d, body %s", rec.Code, rec.Body)
	}
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("operator login: got status %d, body %s", loginRec.Code, loginRec.Body)
	}
	return findCookie(t, loginRec, auth.CookieName)
}

func uploadBackup(t *testing.T, h http.Handler, cookie *http.Cookie, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("backup", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write(data)
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/backup/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func buildBackupZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, contents := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		if _, err := f.Write([]byte(contents)); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zw.Close: %v", err)
	}
	return buf.Bytes()
}

func TestExportBackupRequiresAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	operator := setupOperator(t, h, admin)

	rec := doJSON(t, h, http.MethodGet, "/backup/export", nil, operator)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403 for a non-admin", rec.Code)
	}
}

func TestExportBackupProducesAZipWithTheDataFiles(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, admin)
	doJSON(t, h, http.MethodPost, "/clients", map[string]any{
		"name": "Acme", "code": "ACME", "tagline": []string{"a", "b", "c"}, "logo": validLogoSVG,
	}, admin)

	rec := doJSON(t, h, http.MethodGet, "/backup/export", nil, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Fatalf("got Content-Type %q, want application/zip", ct)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("the response body is not a valid zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"employees.json", "clients.json", "users.json", "logos/acme.svg"} {
		if !names[want] {
			t.Fatalf("expected the backup zip to contain %s, got %v", want, names)
		}
	}
}

func TestImportBackupRequiresAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	operator := setupOperator(t, h, admin)

	data := buildBackupZip(t, map[string]string{"employees.json": "[]", "clients.json": "[]", "users.json": "[]"})
	rec := uploadBackup(t, h, operator, "backup.zip", data)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403 for a non-admin", rec.Code)
	}
}

func TestImportBackupReplacesDataAndSignsEveryoneOut(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, admin)

	hash, salt, err := auth.HashPIN("5678")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	usersJSON, err := json.Marshal([]map[string]any{{
		"username": "restored", "name": "Restored Admin", "role": "admin",
		"pin_hash": hash, "salt": salt,
	}})
	if err != nil {
		t.Fatalf("marshal users.json: %v", err)
	}
	data := buildBackupZip(t, map[string]string{
		"employees.json": `[{"employee_id":"E999","name":"Restored Employee","join_date":"2026-02-01","crop":{"zoom":1,"x":50,"y":50}}]`,
		"clients.json":   `[]`,
		"users.json":     string(usersJSON),
	})
	rec := uploadBackup(t, h, admin, "backup.zip", data)
	if rec.Code != http.StatusOK {
		t.Fatalf("import: got status %d, body %s", rec.Code, rec.Body)
	}

	// The admin's own session must no longer work: users.json changed wholesale.
	meRec := doJSON(t, h, http.MethodGet, "/me", nil, admin)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401 for the pre-import session after import", meRec.Code)
	}

	// The restored user and employee must actually be in place.
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "restored", "pin": "5678"})
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login as restored user: got status %d, body %s", loginRec.Code, loginRec.Body)
	}
	restored := findCookie(t, loginRec, auth.CookieName)

	empRec := doJSON(t, h, http.MethodGet, "/employees", nil, restored)
	var emps []map[string]any
	if err := json.Unmarshal(empRec.Body.Bytes(), &emps); err != nil {
		t.Fatalf("unmarshal employees: %v", err)
	}
	if len(emps) != 1 || emps[0]["employee_id"] != "E999" {
		t.Fatalf("expected the backup's employee E999 to replace the pre-import data, got %+v", emps)
	}
}

func TestImportBackupRejectsAZipMissingRequiredFiles(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	data := buildBackupZip(t, map[string]string{"employees.json": "[]"})
	rec := uploadBackup(t, h, admin, "backup.zip", data)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400 for a backup missing clients.json/users.json", rec.Code)
	}

	// The admin's own session must still work: a rejected import changes nothing.
	meRec := doJSON(t, h, http.MethodGet, "/me", nil, admin)
	if meRec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200 — a rejected import must not sign anyone out", meRec.Code)
	}
}

func TestImportBackupRejectsANonZipFile(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := uploadBackup(t, h, admin, "backup.zip", []byte("not a zip file"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400 for a non-zip upload", rec.Code)
	}
}

func TestClearDataRequiresAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	operator := setupOperator(t, h, admin)

	rec := doJSON(t, h, http.MethodPost, "/backup/clear", nil, operator)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403 for a non-admin", rec.Code)
	}
}

func TestClearDataWipesAndReseedsHeliosAndSignsEveryoneOut(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, admin)

	rec := doJSON(t, h, http.MethodPost, "/backup/clear", nil, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear: got status %d, body %s", rec.Code, rec.Body)
	}

	meRec := doJSON(t, h, http.MethodGet, "/me", nil, admin)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401 after clearing data", meRec.Code)
	}

	setupRec := doJSON(t, h, http.MethodGet, "/setup", nil)
	var status map[string]bool
	if err := json.NewDecoder(setupRec.Body).Decode(&status); err != nil {
		t.Fatalf("decode /setup response: %v", err)
	}
	if !status["needed"] {
		t.Fatal("expected setup to be needed again after clearing data, like a fresh install")
	}

	clients, err := srv.Clients.Store.List()
	if err != nil {
		t.Fatalf("List clients: %v", err)
	}
	if len(clients) != 1 || clients[0].Code != "Helios" {
		t.Fatalf("expected Helios to be reseeded after clearing, got %+v", clients)
	}

	if _, err := os.Stat(filepath.Join(srv.Info.DataDir, "activity.jsonl")); !os.IsNotExist(err) {
		t.Fatal("expected activity.jsonl to be removed by clear")
	}
}
