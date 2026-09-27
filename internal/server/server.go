// Package server wires the HTTP API together: routing, shared middleware
// and JSON helpers used by every handler group.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"idcardstudio/internal/activity"
	"idcardstudio/internal/auth"
	"idcardstudio/internal/clients"
	"idcardstudio/internal/presence"
)

// MaxRequestBody caps every API request body, per the spec's 10 MB limit.
const MaxRequestBody = 10 << 20

// Info is the static server details shown in the "Under the hood" panel.
type Info struct {
	StartedAt time.Time
	Port      int
	DataDir   string
	ExportDir string
}

// Server holds the dependencies shared by all API handlers.
type Server struct {
	Auth     *auth.Manager
	Activity *activity.Log
	Presence *presence.Manager
	Clients  *clients.Manager
	Info     Info
}

// New returns a Server backed by the given auth manager, activity log,
// presence tracker and clients manager.
func New(authManager *auth.Manager, activityLog *activity.Log, presenceManager *presence.Manager, clientsManager *clients.Manager, info Info) *Server {
	return &Server{Auth: authManager, Activity: activityLog, Presence: presenceManager, Clients: clientsManager, Info: info}
}

// Handler builds the complete API mux, with the shared body-size limit and
// X-Requested-With check applied to every route.
func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	s.registerAuthRoutes(api)
	s.registerUserRoutes(api)
	s.registerActivityRoutes(api)
	s.registerPresenceRoutes(api)
	s.registerClientRoutes(api)

	return limitBody(auth.RequireXRequestedWith(api))
}

// logAction records one activity entry, resolving the caller's IP and
// (cached) hostname from r. A write failure only reaches the server log:
// the action it's recording has already succeeded or failed on its own.
func (s *Server) logAction(r *http.Request, username, action, target string) {
	ip := clientIP(r)
	entry := activity.Entry{
		Username: username,
		IP:       ip,
		Host:     s.Presence.ResolveHost(ip),
		Action:   action,
		Target:   target,
	}
	if err := s.Activity.Write(entry); err != nil {
		log.Printf("activity log: %v", err)
	}
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
