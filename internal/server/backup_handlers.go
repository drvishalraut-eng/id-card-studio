package server

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"idcardstudio/internal/auth"
	"idcardstudio/internal/backup"
)

// maxBackupImportSize is a generous ceiling for a whole data/ folder —
// mostly employee photos, each already capped by maxPhotoSize — well above
// the shared 10 MB request-body limit that applies to every other route
// (see limitBody in server.go).
const maxBackupImportSize = 500 << 20

func (s *Server) registerBackupRoutes(mux *http.ServeMux) {
	mux.Handle("GET /backup/export", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleExportBackup)))
	mux.Handle("POST /backup/import", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleImportBackup)))
	mux.Handle("POST /backup/clear", s.Auth.RequireRole(auth.RoleAdmin, http.HandlerFunc(s.handleClearData)))
}

// backupPaths locates every file a backup covers from the pieces the server
// already has: Info.DataDir/PhotosDir come from the portable data/ layout,
// and Clients.LogosDir is the clients manager's own logo folder.
func (s *Server) backupPaths() backup.Paths {
	return backup.Paths{Root: s.Info.DataDir, Logos: s.Clients.LogosDir, Photos: s.Info.PhotosDir}
}

func (s *Server) handleExportBackup(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := backup.Write(&buf, s.backupPaths()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not build the backup: "+err.Error())
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	filename := fmt.Sprintf("idcard-backup-%s.zip", time.Now().Format("2006-01-02-1504"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
	s.logAction(r, actor.Username, "backup_exported", filename)
}

func (s *Server) handleImportBackup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxBackupImportSize); err != nil {
		writeError(w, http.StatusBadRequest, "upload must be multipart/form-data under 500 MB: "+err.Error())
		return
	}
	file, _, err := r.FormFile("backup")
	if err != nil {
		writeError(w, http.StatusBadRequest, "a backup zip file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the uploaded backup: "+err.Error())
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "that file is not a valid zip archive")
		return
	}

	if err := backup.Import(zr, s.backupPaths()); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// users.json (and everything else) just changed wholesale: every
	// existing session, including the admin who triggered this, might now
	// point at a different account or none at all, so sign everyone out.
	s.Auth.Sessions.Clear()

	// Record the restore in the (now-restored) activity log, so it isn't
	// silently invisible in the log that came back with the backup.
	actor, _ := auth.UserFromContext(r.Context())
	s.logAction(r, actor.Username, "backup_imported", "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleClearData(w http.ResponseWriter, r *http.Request) {
	if err := backup.Clear(s.backupPaths()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not clear data: "+err.Error())
		return
	}
	if err := s.Clients.SeedHelios(); err != nil {
		writeError(w, http.StatusInternalServerError, "data cleared but the Helios client could not be reseeded: "+err.Error())
		return
	}
	// users.json is now empty, so every session (including the actor's own)
	// is meaningless — sign everyone out and let the first-run setup screen
	// take over, same as a genuinely fresh install.
	s.Auth.Sessions.Clear()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
