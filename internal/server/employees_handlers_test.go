package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"idcardstudio/internal/auth"
	"idcardstudio/internal/employees"
)

func TestSaveEmployeeCreatesThenUpdates(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "role": "Operator",
		"client": "Helios", "join_date": "2026-01-15",
	}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var e employees.Employee
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.UpdatedBy != "ada" {
		t.Fatalf("got UpdatedBy=%q", e.UpdatedBy)
	}

	listRec := doJSON(t, h, http.MethodGet, "/employees", nil, admin)
	var list []employees.Employee
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d employees, want 1", len(list))
	}
}

func TestSaveEmployeeAllowsOperator(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	opal := findCookie(t, loginRec, auth.CookieName)

	rec := doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, opal)
	if rec.Code != http.StatusOK {
		t.Fatalf("operators should be able to add employees, got status %d, body %s", rec.Code, rec.Body)
	}
}

func TestSaveEmployeeRejectsBadInput(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "bad id!", "name": "X", "join_date": "2026-01-15",
	}, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestImportEmployeesReportsSkippedRows(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/employees/import", map[string]any{
		"rows": []map[string]any{
			{"employee_id": "E001", "name": "Priya Rao", "client": "Helios", "join_date": "2026-01-15"},
			{"employee_id": "", "name": "No ID", "join_date": "2026-01-15"},
		},
	}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var result employees.ImportResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Imported != 1 || len(result.Skipped) != 1 {
		t.Fatalf("got %+v", result)
	}
}

func TestMapClientEndpoint(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "client": "helioss", "join_date": "2026-01-15",
	}, admin)

	rec := doJSON(t, h, http.MethodPost, "/employees/E001/client", map[string]string{"client": "Helios"}, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var e employees.Employee
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.ClientCode != "Helios" {
		t.Fatalf("got ClientCode=%q", e.ClientCode)
	}
}

// validJPEG is a 1x1 pixel JPEG, small enough to embed as a test fixture.
var validJPEG = []byte{
	0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
}

func TestUploadPhotoSavesFileAndUpdatesCrop(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, admin)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("photo", "photo.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write(validJPEG)
	mw.WriteField("zoom", "1.2")
	mw.WriteField("x", "40")
	mw.WriteField("y", "60")
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/employees/E001/photo", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	req.AddCookie(admin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var e employees.Employee
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Photo != "E001.jpg" {
		t.Fatalf("got Photo=%q", e.Photo)
	}
	if e.Crop.Zoom != 1.2 || e.Crop.X != 40 || e.Crop.Y != 60 {
		t.Fatalf("got Crop=%+v", e.Crop)
	}
}

func TestUploadPhotoRejectsNonJPEG(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-01-15",
	}, admin)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("photo", "photo.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write([]byte("not a jpeg"))
	mw.WriteField("zoom", "1")
	mw.WriteField("x", "50")
	mw.WriteField("y", "50")
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/employees/E001/photo", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	req.AddCookie(admin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestUploadPhotoRejectsUnknownEmployee(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("photo", "photo.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write(validJPEG)
	mw.WriteField("zoom", "1")
	mw.WriteField("x", "50")
	mw.WriteField("y", "50")
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/employees/nope/photo", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	req.AddCookie(admin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", rec.Code)
	}
}
