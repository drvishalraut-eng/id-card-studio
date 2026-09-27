package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"idcardstudio/internal/auth"
)

const testPDF = "%PDF-1.4\nfake-body\n%%EOF"

func postExport(t *testing.T, h http.Handler, employeeID string, pdf []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("employee_id", employeeID); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	part, err := mw.CreateFormFile("pdf", "card.pdf")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write(pdf)
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/export", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestExportSavesPDFUnderMonthFolder(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	admin := setupAdmin(t, h)

	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "client": "Helios", "join_date": "2026-09-15",
	}, admin)

	rec := postExport(t, h, "E001", []byte(testPDF), admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}

	wantPath := filepath.Join(srv.Info.ExportDir, "Sep 2026", "E001_Priya_Rao.pdf")
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", wantPath, err)
	}
	if string(data) != testPDF {
		t.Fatal("saved PDF contents do not match")
	}
}

func TestExportRecordsExportedAtAndBy(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-09-15",
	}, admin)

	postExport(t, h, "E001", []byte(testPDF), admin)

	e, found, err := srv.Employees.Store.Find("E001")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if !found || e.ExportedAt == nil || e.ExportedBy != "ada" {
		t.Fatalf("got %+v, found=%v", e, found)
	}
}

func TestExportRequiresAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	req := httptest.NewRequest(http.MethodPost, "/export", nil)
	req.Header.Set(auth.RequestedWithHeader, auth.RequestedWithValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestExportRejectsUnknownEmployee(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	rec := postExport(t, h, "nope", []byte(testPDF), admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}

func TestExportRejectsNonPDF(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/employees", map[string]any{
		"employee_id": "E001", "name": "Priya Rao", "join_date": "2026-09-15",
	}, admin)

	rec := postExport(t, h, "E001", []byte("not a pdf"), admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}
