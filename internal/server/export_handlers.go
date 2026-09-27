package server

import (
	"io"
	"net/http"

	"idcardstudio/internal/auth"
)

// maxCardPDFSize is generous headroom for a two-page, 600 DPI CR80 card
// PDF; the shared 10 MB request-body limit already applies ahead of this.
const maxCardPDFSize = 8 << 20

func (s *Server) registerExportRoutes(mux *http.ServeMux) {
	mux.Handle("POST /export", s.Auth.RequireAuth(http.HandlerFunc(s.handleExport)))
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxCardPDFSize); err != nil {
		writeError(w, http.StatusBadRequest, "upload must be multipart/form-data under 8 MB: "+err.Error())
		return
	}
	employeeID := r.FormValue("employee_id")
	if employeeID == "" {
		writeError(w, http.StatusBadRequest, "employee_id is required")
		return
	}
	file, _, err := r.FormFile("pdf")
	if err != nil {
		writeError(w, http.StatusBadRequest, "a rendered pdf file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCardPDFSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the uploaded PDF: "+err.Error())
		return
	}
	if len(data) > maxCardPDFSize {
		writeError(w, http.StatusBadRequest, "PDF is too large (max 8 MB)")
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	result, err := s.Export.Save(employeeID, data, actor.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logAction(r, actor.Username, "employee_exported", result.EmployeeID)
	writeJSON(w, http.StatusOK, map[string]string{"employee_id": result.EmployeeID})
}
