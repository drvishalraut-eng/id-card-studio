package auth

import (
	"fmt"
	"strings"
	"time"

	"idcardstudio/internal/storage"
)

// Role is a user's permission level.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
)

// User is one row of users.json. PINHash and Salt are base64-encoded PBKDF2
// output, never the raw PIN.
type User struct {
	Username       string     `json:"username"`
	Name           string     `json:"name"`
	Role           Role       `json:"role"`
	PINHash        string     `json:"pin_hash"`
	Salt           string     `json:"salt"`
	Disabled       bool       `json:"disabled"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`
}

// UserStore is the mutex-protected, atomically-written users.json file.
type UserStore struct {
	store *storage.Store[[]User]
}

// NewUserStore opens (or prepares to create) the users.json file at path.
func NewUserStore(path string) *UserStore {
	return &UserStore{store: storage.New[[]User](path)}
}

// List returns every user, in no particular order. Never nil — see
// employees.Store.List's comment for why that matters for the API.
func (s *UserStore) List() ([]User, error) {
	list, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []User{}
	}
	return list, nil
}

// Find returns the user with the given username (case-insensitive), if any.
func (s *UserStore) Find(username string) (User, bool, error) {
	users, err := s.store.Load()
	if err != nil {
		return User{}, false, err
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			return u, true, nil
		}
	}
	return User{}, false, nil
}

// Create adds a new user, rejecting a username that is already taken
// (case-insensitively).
func (s *UserStore) Create(u User) error {
	_, err := s.store.Update(func(users []User) ([]User, error) {
		for _, existing := range users {
			if strings.EqualFold(existing.Username, u.Username) {
				return users, fmt.Errorf("username %q is already taken", u.Username)
			}
		}
		return append(users, u), nil
	})
	return err
}

// Update replaces the stored user matching username with the value returned
// by fn, atomically. fn receives the current stored value.
func (s *UserStore) Update(username string, fn func(User) (User, error)) (User, error) {
	var updated User
	_, err := s.store.Update(func(users []User) ([]User, error) {
		for i, u := range users {
			if strings.EqualFold(u.Username, username) {
				next, err := fn(u)
				if err != nil {
					return users, err
				}
				users[i] = next
				updated = next
				return users, nil
			}
		}
		return users, fmt.Errorf("no user %q", username)
	})
	return updated, err
}
