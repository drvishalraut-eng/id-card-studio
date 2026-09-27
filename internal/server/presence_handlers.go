package server

import (
	"encoding/json"
	"net/http"
	"time"

	"idcardstudio/internal/auth"
	"idcardstudio/internal/netutil"
	"idcardstudio/internal/presence"
)

func (s *Server) registerPresenceRoutes(mux *http.ServeMux) {
	mux.Handle("POST /presence", s.Auth.RequireAuth(http.HandlerFunc(s.handlePresenceBeat)))
	mux.Handle("GET /server-info", s.Auth.RequireAuth(http.HandlerFunc(s.handleServerInfo)))
}

type presenceBeatRequest struct {
	Step string `json:"step"`
}

func (s *Server) handlePresenceBeat(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	var req presenceBeatRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // step is a display hint; a bad body just means no step

	s.Presence.Beat(u.Username, clientIP(r), req.Step)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type serverInfoResponse struct {
	You       string           `json:"you"`
	Users     []presence.Entry `json:"users"`
	URLs      []string         `json:"urls"`
	Uptime    string           `json:"uptime"`
	DataDir   string           `json:"data_dir"`
	ExportDir string           `json:"export_dir"`
}

func (s *Server) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, serverInfoResponse{
		You:       u.Username,
		Users:     s.Presence.List(),
		URLs:      netutil.LANURLs(s.Info.Port),
		Uptime:    time.Since(s.Info.StartedAt).Round(time.Second).String(),
		DataDir:   s.Info.DataDir,
		ExportDir: s.Info.ExportDir,
	})
}
