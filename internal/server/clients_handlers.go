package server

import (
	"encoding/json"
	"net/http"

	"idcardstudio/internal/auth"
)

func (s *Server) registerClientRoutes(mux *http.ServeMux) {
	mux.Handle("GET /clients", s.Auth.RequireAuth(http.HandlerFunc(s.handleListClients)))
	mux.Handle("POST /clients", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleAddClient)))
	mux.Handle("PUT /clients/{id}", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleUpdateClient)))
}

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	list, err := s.Clients.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load clients: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type clientRequest struct {
	Name    string   `json:"name"`
	Code    string   `json:"code"`
	Logo    string   `json:"logo"`
	Tagline []string `json:"tagline"`
}

func (s *Server) handleAddClient(w http.ResponseWriter, r *http.Request) {
	var req clientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	c, err := s.Clients.Create(req.Name, req.Code, req.Tagline, req.Logo)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		s.logAction(r, actor.Username, "client_added", c.Name)
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req clientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	c, err := s.Clients.Update(id, req.Name, req.Code, req.Tagline, req.Logo)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		s.logAction(r, actor.Username, "client_updated", c.Name)
	}
	writeJSON(w, http.StatusOK, c)
}
