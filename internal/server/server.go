// Package server wires the HTTP API together: routing, shared middleware
// and JSON helpers used by every handler group.
package server

import (
	"encoding/json"
	"net/http"

	"idcardstudio/internal/auth"
)

// MaxRequestBody caps every API request body, per the spec's 10 MB limit.
const MaxRequestBody = 10 << 20

// Server holds the dependencies shared by all API handlers.
type Server struct {
	Auth *auth.Manager
}

// New returns a Server backed by the given auth manager.
func New(authManager *auth.Manager) *Server {
	return &Server{Auth: authManager}
}

// Handler builds the complete API mux, with the shared body-size limit and
// X-Requested-With check applied to every route.
func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	s.registerAuthRoutes(api)

	return limitBody(auth.RequireXRequestedWith(api))
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBody)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
