package auth

import "errors"

var (
	ErrInvalidRole      = errors.New(`role must be "admin" or "operator"`)
	ErrLastActiveAdmin  = errors.New("can't disable the last active admin")
	ErrUsernameRequired = errors.New("username is required")
	ErrNameRequired     = errors.New("name is required")
)

// AddUser creates a new user with the given role and PIN. It is the
// Admin-only "add user" action on the Users page.
func (m *Manager) AddUser(name, username string, role Role, pin string) (User, error) {
	if name == "" {
		return User{}, ErrNameRequired
	}
	if username == "" {
		return User{}, ErrUsernameRequired
	}
	if role != RoleAdmin && role != RoleOperator {
		return User{}, ErrInvalidRole
	}
	if err := ValidatePINFormat(pin); err != nil {
		return User{}, err
	}

	hash, salt, err := HashPIN(pin)
	if err != nil {
		return User{}, err
	}
	u := User{Username: username, Name: name, Role: role, PINHash: hash, Salt: salt}
	if err := m.Users.Create(u); err != nil {
		return User{}, err
	}
	return u, nil
}

// ResetPIN sets a new PIN for username and clears any lockout, as the
// Admin-only "reset PIN" action.
func (m *Manager) ResetPIN(username, pin string) (User, error) {
	if err := ValidatePINFormat(pin); err != nil {
		return User{}, err
	}
	hash, salt, err := HashPIN(pin)
	if err != nil {
		return User{}, err
	}
	return m.Users.Update(username, func(u User) (User, error) {
		u.PINHash = hash
		u.Salt = salt
		u.FailedAttempts = 0
		u.LockedUntil = nil
		return u, nil
	})
}

// SetDisabled enables or disables username. Disabling is rejected if
// username is the last active (non-disabled) Admin.
func (m *Manager) SetDisabled(username string, disabled bool) (User, error) {
	if disabled {
		users, err := m.Users.List()
		if err != nil {
			return User{}, err
		}
		activeAdmins := 0
		for _, u := range users {
			if u.Role == RoleAdmin && !u.Disabled {
				activeAdmins++
			}
		}
		target, ok, err := m.Users.Find(username)
		if err != nil {
			return User{}, err
		}
		if ok && target.Role == RoleAdmin && !target.Disabled && activeAdmins <= 1 {
			return User{}, ErrLastActiveAdmin
		}
	}
	return m.Users.Update(username, func(u User) (User, error) {
		u.Disabled = disabled
		return u, nil
	})
}
