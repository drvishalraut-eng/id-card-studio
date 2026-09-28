// Package auth handles PIN hashing, sessions, lockout and role checks.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// MaxFailedAttempts is how many wrong PINs in a row lock an account.
const MaxFailedAttempts = 5

// LockoutDuration is how long a locked account stays locked.
const LockoutDuration = 5 * time.Minute

// CookieName is the session cookie set on login and cleared on logout.
const CookieName = "idcard_session"

// RequestedWithHeader and RequestedWithValue must be present on every API
// call, as a lightweight cross-site request check.
const RequestedWithHeader = "X-Requested-With"
const RequestedWithValue = "idcard"

var (
	ErrAlreadySetUp       = errors.New("setup has already been completed; sign in instead")
	ErrInvalidCredentials = errors.New("username or PIN is incorrect")
	ErrAccountDisabled    = errors.New("this account has been disabled")
	ErrAccountLocked      = errors.New("too many failed attempts; try again in a few minutes")
)

// Manager wires together the user store and session store to implement
// setup, login, logout and role-checked middleware.
type Manager struct {
	Users    *UserStore
	Sessions *SessionManager
}

// NewManager returns a Manager backed by the given users.json path.
func NewManager(usersPath string) *Manager {
	return &Manager{
		Users:    NewUserStore(usersPath),
		Sessions: NewSessionManager(),
	}
}

// NeedsSetup reports whether no users exist yet, meaning the first-run
// setup screen should be shown.
func (m *Manager) NeedsSetup() (bool, error) {
	users, err := m.Users.List()
	if err != nil {
		return false, err
	}
	return len(users) == 0, nil
}

// Setup creates the first Admin account. It fails if any user already
// exists or if name, username or pin are invalid.
func (m *Manager) Setup(name, username, pin string) (User, error) {
	needsSetup, err := m.NeedsSetup()
	if err != nil {
		return User{}, err
	}
	if !needsSetup {
		return User{}, ErrAlreadySetUp
	}
	if name == "" {
		return User{}, errors.New("name is required")
	}
	if username == "" {
		return User{}, errors.New("username is required")
	}
	if err := ValidatePINFormat(pin); err != nil {
		return User{}, err
	}

	hash, salt, err := HashPIN(pin)
	if err != nil {
		return User{}, err
	}
	u := User{
		Username: username,
		Name:     name,
		Role:     RoleAdmin,
		PINHash:  hash,
		Salt:     salt,
	}
	if err := m.Users.Create(u); err != nil {
		return User{}, err
	}
	return u, nil
}

// Login verifies username and pin, applying lockout rules, and returns a
// new session token plus the signed-in user on success.
func (m *Manager) Login(username, pin string) (token string, user User, err error) {
	u, ok, err := m.Users.Find(username)
	if err != nil {
		return "", User{}, err
	}
	if !ok {
		return "", User{}, ErrInvalidCredentials
	}
	if u.Disabled {
		return "", User{}, ErrAccountDisabled
	}
	if u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) {
		return "", User{}, ErrAccountLocked
	}

	if !VerifyPIN(pin, u.PINHash, u.Salt) {
		locked, updateErr := m.Users.Update(u.Username, func(cur User) (User, error) {
			cur.FailedAttempts++
			if cur.FailedAttempts >= MaxFailedAttempts {
				until := time.Now().Add(LockoutDuration)
				cur.LockedUntil = &until
				cur.FailedAttempts = 0
			}
			return cur, nil
		})
		if updateErr != nil {
			return "", User{}, updateErr
		}
		if locked.LockedUntil != nil && time.Now().Before(*locked.LockedUntil) {
			return "", User{}, ErrAccountLocked
		}
		return "", User{}, ErrInvalidCredentials
	}

	now := time.Now()
	signedIn, err := m.Users.Update(u.Username, func(cur User) (User, error) {
		cur.FailedAttempts = 0
		cur.LockedUntil = nil
		cur.LastLogin = &now
		return cur, nil
	})
	if err != nil {
		return "", User{}, err
	}

	token, err = m.Sessions.Create(signedIn.Username)
	if err != nil {
		return "", User{}, err
	}
	return token, signedIn, nil
}

// Logout ends the session identified by token.
func (m *Manager) Logout(token string) {
	m.Sessions.Delete(token)
}

// CurrentUser resolves the session cookie on r to its signed-in, non-disabled
// user.
func (m *Manager) CurrentUser(r *http.Request) (User, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return User{}, false
	}
	username, ok := m.Sessions.Username(cookie.Value)
	if !ok {
		return User{}, false
	}
	u, ok, err := m.Users.Find(username)
	if err != nil || !ok || u.Disabled {
		return User{}, false
	}
	return u, true
}

// SetSessionCookie attaches a session cookie for token to w.
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(SessionDuration.Seconds()),
	})
}

// ClearSessionCookie removes the session cookie from the browser.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

type contextKey int

const userContextKey contextKey = iota

// UserFromContext returns the signed-in user attached by RequireAuth.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userContextKey).(User)
	return u, ok
}

// RequireXRequestedWith rejects any request missing the required
// X-Requested-With: idcard header, as a lightweight cross-site request check.
//
// A GET photo request is exempt: it's loaded by a plain <img src>, which the
// browser never attaches custom headers to, so requiring the header there
// would make every saved photo unloadable. It's still gated by RequireAuth's
// SameSite=Strict session cookie, which a cross-site <img> can't carry either.
func RequireXRequestedWith(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPhotoGet(r) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get(RequestedWithHeader) != RequestedWithValue {
			writeError(w, http.StatusBadRequest, "missing X-Requested-With: idcard header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isPhotoGet(r *http.Request) bool {
	return r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/employees/") && strings.HasSuffix(r.URL.Path, "/photo")
}

// RequireAuth rejects requests without a valid session and otherwise attaches
// the signed-in user to the request context for downstream handlers.
func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := m.CurrentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "sign in to continue")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is like RequireAuth but also rejects a signed-in user whose
// role is not role.
func (m *Manager) RequireRole(role Role, next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := UserFromContext(r.Context())
		if u.Role != role {
			writeError(w, http.StatusForbidden, "this action requires the "+string(role)+" role")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
