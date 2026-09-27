package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"idcardstudio/internal/auth"
	"idcardstudio/internal/clients"
)

func TestListClientsAllowsAnySignedInUser(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	opal := findCookie(t, loginRec, auth.CookieName)

	rec := doJSON(t, h, http.MethodGet, "/clients", nil, opal)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
}

func TestListClientsRequiresAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodGet, "/clients", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestAddClientRequiresAdmin(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)
	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "opal", "pin": "2222"})
	opal := findCookie(t, loginRec, auth.CookieName)

	rec := doJSON(t, h, http.MethodPost, "/clients", map[string]any{
		"name": "Acme", "code": "ACME", "tagline": []string{"a", "b", "c"}, "logo": validLogoSVG,
	}, opal)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want 403", rec.Code)
	}
}

const validLogoSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 40"><path d="M0 0h100v40h-100z" fill="#fff"/></svg>`

func TestAddThenListThenUpdateClient(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	addRec := doJSON(t, h, http.MethodPost, "/clients", map[string]any{
		"name": "Acme", "code": "ACME", "tagline": []string{"a", "b", "c"}, "logo": validLogoSVG,
	}, admin)
	if addRec.Code != http.StatusCreated {
		t.Fatalf("add: got status %d, body %s", addRec.Code, addRec.Body)
	}
	var created clients.Client
	if err := json.Unmarshal(addRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	listRec := doJSON(t, h, http.MethodGet, "/clients", nil, admin)
	var list []clients.Client
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("got %+v", list)
	}

	updateRec := doJSON(t, h, http.MethodPut, "/clients/"+created.ID, map[string]any{
		"name": "Acme Corp", "code": "ACMECORP", "tagline": []string{"x", "y", "z"},
	}, admin)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update: got status %d, body %s", updateRec.Code, updateRec.Body)
	}
	var updated clients.Client
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updated.Name != "Acme Corp" || updated.Logo != validLogoSVG {
		t.Fatalf("got %+v", updated)
	}
}

func TestAddClientRejectsInvalidSVG(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodPost, "/clients", map[string]any{
		"name": "Acme", "code": "ACME", "tagline": []string{"a", "b", "c"}, "logo": "<not-svg/>",
	}, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400", rec.Code)
	}
}
