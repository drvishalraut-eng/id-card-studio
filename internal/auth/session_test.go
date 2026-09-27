package auth

import (
	"testing"
	"time"
)

func TestSessionCreateThenUsername(t *testing.T) {
	sm := NewSessionManager()
	token, err := sm.Create("alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	username, ok := sm.Username(token)
	if !ok || username != "alice" {
		t.Fatalf("got (%q, %v), want (alice, true)", username, ok)
	}
}

func TestSessionUnknownTokenIsRejected(t *testing.T) {
	sm := NewSessionManager()
	if _, ok := sm.Username("nonexistent"); ok {
		t.Fatal("expected an unknown token to be rejected")
	}
}

func TestSessionDeleteEndsIt(t *testing.T) {
	sm := NewSessionManager()
	token, _ := sm.Create("alice")
	sm.Delete(token)
	if _, ok := sm.Username(token); ok {
		t.Fatal("expected the session to be gone after Delete")
	}
}

func TestSessionExpires(t *testing.T) {
	sm := NewSessionManager()
	token, _ := sm.Create("alice")
	sm.mu.Lock()
	s := sm.sessions[token]
	s.ExpiresAt = time.Now().Add(-time.Second)
	sm.sessions[token] = s
	sm.mu.Unlock()

	if _, ok := sm.Username(token); ok {
		t.Fatal("expected an expired session to be rejected")
	}
}

func TestSessionTokensAreUnique(t *testing.T) {
	sm := NewSessionManager()
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		token, err := sm.Create("alice")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if seen[token] {
			t.Fatalf("duplicate token generated: %q", token)
		}
		seen[token] = true
	}
}
