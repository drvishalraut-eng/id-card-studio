package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"idcardstudio/internal/auth"
)

func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /setup", s.handleSetupStatus)
	mux.HandleFunc("POST /setup", s.handleSetup)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("GET /me", s.handleMe)
}

type userResponse struct {
	Username string    `json:"username"`
	Name     string    `json:"name"`
	Role     auth.Role `json:"role"`
}

func toUserResponse(u auth.User) userResponse {
	return userResponse{Username: u.Username, Name: u.Name, Role: u.Role}
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	needed, err := s.Auth.NeedsSetup()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check setup status: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"needed": needed})
}

type setupRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	PIN      string `json:"pin"`
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	u, err := s.Auth.Setup(req.Name, req.Username, req.PIN)
	if err != nil {
		if errors.Is(err, auth.ErrAlreadySetUp) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, _, err := s.Auth.Login(req.Username, req.PIN)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "account was created but sign-in failed: "+err.Error())
		return
	}
	s.logAction(r, u.Username, "user_added", u.Username)
	s.logAction(r, u.Username, "sign_in", u.Username)
	auth.SetSessionCookie(w, token)
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

type loginRequest struct {
	Username string `json:"username"`
	PIN      string `json:"pin"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	token, u, err := s.Auth.Login(req.Username, req.PIN)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrAccountLocked):
			writeError(w, http.StatusLocked, err.Error())
		case errors.Is(err, auth.ErrAccountDisabled):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, err.Error())
			s.logAction(r, req.Username, "sign_in_failed", req.Username)
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	s.logAction(r, u.Username, "sign_in", u.Username)
	auth.SetSessionCookie(w, token)
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	u, hadSession := s.Auth.CurrentUser(r)
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		s.Auth.Logout(cookie.Value)
	}
	if hadSession {
		s.logAction(r, u.Username, "sign_out", u.Username)
	}
	auth.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := s.Auth.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "sign in to continue")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}
