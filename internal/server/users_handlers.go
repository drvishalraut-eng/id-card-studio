package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"idcardstudio/internal/auth"
)

func (s *Server) registerUserRoutes(mux *http.ServeMux) {
	mux.Handle("GET /users", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleListUsers)))
	mux.Handle("POST /users", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleAddUser)))
	mux.Handle("POST /users/{username}/reset-pin", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleResetPIN)))
	mux.Handle("POST /users/{username}/disable", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleSetDisabled(true))))
	mux.Handle("POST /users/{username}/enable", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleSetDisabled(false))))
}

type userListItem struct {
	Username  string     `json:"username"`
	Name      string     `json:"name"`
	Role      auth.Role  `json:"role"`
	Status    string     `json:"status"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

func toUserListItem(u auth.User) userListItem {
	status := "active"
	switch {
	case u.Disabled:
		status = "disabled"
	case u.LockedUntil != nil && time.Now().Before(*u.LockedUntil):
		status = "locked"
	}
	return userListItem{Username: u.Username, Name: u.Name, Role: u.Role, Status: status, LastLogin: u.LastLogin}
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Auth.Users.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load users: "+err.Error())
		return
	}
	items := make([]userListItem, len(users))
	for i, u := range users {
		items[i] = toUserListItem(u)
	}
	writeJSON(w, http.StatusOK, items)
}

type addUserRequest struct {
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Role       auth.Role `json:"role"`
	PIN        string    `json:"pin"`
	ConfirmPIN string    `json:"confirm_pin"`
}

func (s *Server) handleAddUser(w http.ResponseWriter, r *http.Request) {
	var req addUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if req.PIN != req.ConfirmPIN {
		writeError(w, http.StatusBadRequest, "PIN and confirm PIN must match")
		return
	}

	u, err := s.Auth.AddUser(req.Name, req.Username, req.Role, req.PIN)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		s.logAction(r, actor.Username, "user_added", u.Username)
	}
	writeJSON(w, http.StatusCreated, toUserListItem(u))
}

type resetPINRequest struct {
	PIN        string `json:"pin"`
	ConfirmPIN string `json:"confirm_pin"`
}

func (s *Server) handleResetPIN(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	var req resetPINRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if req.PIN != req.ConfirmPIN {
		writeError(w, http.StatusBadRequest, "PIN and confirm PIN must match")
		return
	}

	u, err := s.Auth.ResetPIN(username, req.PIN)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		s.logAction(r, actor.Username, "user_pin_reset", u.Username)
	}
	writeJSON(w, http.StatusOK, toUserListItem(u))
}

func (s *Server) handleSetDisabled(disabled bool) http.HandlerFunc {
	action := "user_enabled"
	if disabled {
		action = "user_disabled"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		u, err := s.Auth.SetDisabled(username, disabled)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, auth.ErrLastActiveAdmin) {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		if actor, ok := auth.UserFromContext(r.Context()); ok {
			s.logAction(r, actor.Username, action, u.Username)
		}
		writeJSON(w, http.StatusOK, toUserListItem(u))
	}
}
