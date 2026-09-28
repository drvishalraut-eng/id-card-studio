package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// SessionDuration is how long a session cookie stays valid after login.
const SessionDuration = 12 * time.Hour

type session struct {
	Username  string
	ExpiresAt time.Time
}

// SessionManager holds active sessions in memory, keyed by an opaque token.
// Sessions do not survive a server restart: on a portable, LAN-hosted app
// that's the simplest option the spec allows, and it just means everyone
// signs back in after a restart.
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]session
}

// NewSessionManager returns an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]session)}
}

// Create starts a new session for username and returns its token.
func (m *SessionManager) Create(username string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = session{Username: username, ExpiresAt: time.Now().Add(SessionDuration)}
	return token, nil
}

// Username returns the session's username if token is valid and unexpired.
func (m *SessionManager) Username(token string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return "", false
	}
	if time.Now().After(s.ExpiresAt) {
		delete(m.sessions, token)
		return "", false
	}
	return s.Username, true
}

// Delete ends a session, e.g. on logout.
func (m *SessionManager) Delete(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// Clear ends every active session at once — used after a bulk operation
// that replaces or wipes users.json (importing or clearing a backup), since
// an existing token's username might now belong to a different account or
// no account at all.
func (m *SessionManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]session)
}
