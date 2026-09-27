package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPresenceBeatRequiresAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodPost, "/presence", map[string]string{"step": "clients"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestPresenceBeatThenServerInfoShowsUser(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	beatRec := doJSON(t, h, http.MethodPost, "/presence", map[string]string{"step": "employees"}, admin)
	if beatRec.Code != http.StatusOK {
		t.Fatalf("beat: got status %d, body %s", beatRec.Code, beatRec.Body)
	}

	infoRec := doJSON(t, h, http.MethodGet, "/server-info", nil, admin)
	if infoRec.Code != http.StatusOK {
		t.Fatalf("server-info: got status %d, body %s", infoRec.Code, infoRec.Body)
	}
	var info serverInfoResponse
	if err := json.Unmarshal(infoRec.Body.Bytes(), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.You != "ada" {
		t.Fatalf("got You=%q, want ada", info.You)
	}
	if len(info.Users) != 1 || info.Users[0].Username != "ada" || info.Users[0].Step != "employees" {
		t.Fatalf("got Users=%+v", info.Users)
	}
	if info.DataDir == "" || info.ExportDir == "" {
		t.Fatalf("expected DataDir and ExportDir to be set, got %+v", info)
	}
}

func TestServerInfoRequiresAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodGet, "/server-info", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}
