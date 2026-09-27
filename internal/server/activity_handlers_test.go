package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"idcardstudio/internal/activity"
	"idcardstudio/internal/auth"
)

func TestActivityRequiresAuth(t *testing.T) {
	h := newTestServer(t).Handler()
	rec := doJSON(t, h, http.MethodGet, "/activity", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", rec.Code)
	}
}

func TestActivityLogsSignInAndSignOut(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	rec := doJSON(t, h, http.MethodGet, "/activity", nil, admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, body %s", rec.Code, rec.Body)
	}
	var result activity.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var sawUserAdded, sawSignIn bool
	for _, e := range result.Entries {
		if e.Username != "ada" {
			t.Fatalf("unexpected entry actor %q", e.Username)
		}
		switch e.Action {
		case "user_added":
			sawUserAdded = true
		case "sign_in":
			sawSignIn = true
		}
	}
	if !sawUserAdded || !sawSignIn {
		t.Fatalf("got entries %+v, want user_added and sign_in", result.Entries)
	}

	doJSON(t, h, http.MethodPost, "/logout", nil, admin)

	loginRec := doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "ada", "pin": "1234"})
	newCookie := findCookie(t, loginRec, auth.CookieName)

	rec2 := doJSON(t, h, http.MethodGet, "/activity", nil, newCookie)
	var result2 activity.Result
	if err := json.Unmarshal(rec2.Body.Bytes(), &result2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var signOuts, signIns int
	for _, e := range result2.Entries {
		switch e.Action {
		case "sign_out":
			signOuts++
		case "sign_in":
			signIns++
		}
	}
	if signOuts != 1 || signIns != 2 {
		t.Fatalf("got signOuts=%d signIns=%d, want 1 and 2", signOuts, signIns)
	}
}

func TestActivityLogsFailedSignIn(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	doJSON(t, h, http.MethodPost, "/login", map[string]string{"username": "ada", "pin": "0000"})

	rec := doJSON(t, h, http.MethodGet, "/activity?search=sign_in_failed", nil, admin)
	var result activity.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("got %d matching entries, want 1", result.Total)
	}
}

func TestActivityLogsUserManagementActions(t *testing.T) {
	h := newTestServer(t).Handler()
	admin := setupAdmin(t, h)

	doJSON(t, h, http.MethodPost, "/users", map[string]string{
		"name": "Opal Operator", "username": "opal", "role": "operator", "pin": "2222", "confirm_pin": "2222",
	}, admin)
	doJSON(t, h, http.MethodPost, "/users/opal/reset-pin", map[string]string{"pin": "9999", "confirm_pin": "9999"}, admin)
	doJSON(t, h, http.MethodPost, "/users/opal/disable", nil, admin)
	doJSON(t, h, http.MethodPost, "/users/opal/enable", nil, admin)

	rec := doJSON(t, h, http.MethodGet, "/activity?username=ada&page_size=50", nil, admin)
	var result activity.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	wantActions := map[string]bool{"user_pin_reset": false, "user_disabled": false, "user_enabled": false}
	for _, e := range result.Entries {
		if e.Target == "opal" {
			if _, ok := wantActions[e.Action]; ok {
				wantActions[e.Action] = true
			}
		}
	}
	for action, seen := range wantActions {
		if !seen {
			t.Fatalf("expected an activity entry for action %q targeting opal", action)
		}
	}
}
