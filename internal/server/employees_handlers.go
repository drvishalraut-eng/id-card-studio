package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"idcardstudio/internal/auth"
	"idcardstudio/internal/employees"
	"idcardstudio/internal/storage"
)

// maxPhotoSize matches the client-side resize target (long side <= 800px,
// JPEG quality 0.9) with generous headroom; the shared 10 MB request-body
// limit already applies ahead of this.
const maxPhotoSize = 5 << 20

func (s *Server) registerEmployeeRoutes(mux *http.ServeMux) {
	mux.Handle("GET /employees", s.Auth.RequireAuth(http.HandlerFunc(s.handleListEmployees)))
	mux.Handle("POST /employees", s.Auth.RequireAuth(http.HandlerFunc(s.handleSaveEmployee)))
	mux.Handle("POST /employees/import", s.Auth.RequireAuth(http.HandlerFunc(s.handleImportEmployees)))
	mux.Handle("POST /employees/{id}/client", s.Auth.RequireAuth(http.HandlerFunc(s.handleMapClient)))
	mux.Handle("POST /employees/{id}/photo", s.Auth.RequireAuth(http.HandlerFunc(s.handleUploadPhoto)))
	mux.Handle("GET /employees/{id}/photo", s.Auth.RequireAuth(http.HandlerFunc(s.handleGetPhoto)))
	mux.Handle("DELETE /employees/{id}", s.Auth.RequireAuth(http.HandlerFunc(s.handleDeleteEmployee)))
}

func (s *Server) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	list, err := s.Employees.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load employees: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type employeeRequest struct {
	EmployeeID string `json:"employee_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Client     string `json:"client"`
	JoinDate   any    `json:"join_date"`
	PhotoNote  string `json:"photo_note"`
}

func (s *Server) handleSaveEmployee(w http.ResponseWriter, r *http.Request) {
	var req employeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	e, err := s.Employees.Save(employees.Input{
		EmployeeID: req.EmployeeID,
		Name:       req.Name,
		Role:       req.Role,
		Client:     req.Client,
		JoinDate:   req.JoinDate,
		PhotoNote:  req.PhotoNote,
	}, actor.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logAction(r, actor.Username, "employee_saved", e.EmployeeID)
	writeJSON(w, http.StatusOK, e)
}

type importRequest struct {
	Rows []employees.ImportRow `json:"rows"`
}

func (s *Server) handleImportEmployees(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	result, err := s.Employees.Import(req.Rows, actor.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "import failed: "+err.Error())
		return
	}
	s.logAction(r, actor.Username, "employees_imported", fmt.Sprintf("%d imported, %d skipped", result.Imported, len(result.Skipped)))
	writeJSON(w, http.StatusOK, result)
}

type mapClientRequest struct {
	Client string `json:"client"`
}

func (s *Server) handleMapClient(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req mapClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	e, err := s.Employees.MapClient(id, req.Client, actor.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.logAction(r, actor.Username, "employee_client_mapped", e.EmployeeID)
	writeJSON(w, http.StatusOK, e)
}

// handleDeleteEmployee removes an employee entirely, along with its photo
// file if it had one. employee_id can't be edited in place (see
// employees.Manager.Delete) — this is how a wrong ID gets corrected.
func (s *Server) handleDeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	removed, err := s.Employees.Delete(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "no employee with that ID")
		return
	}
	if removed.Photo != "" {
		if err := os.Remove(filepath.Join(s.Info.PhotosDir, removed.Photo)); err != nil && !os.IsNotExist(err) {
			writeError(w, http.StatusInternalServerError, "employee deleted but its photo could not be removed: "+err.Error())
			return
		}
	}

	actor, _ := auth.UserFromContext(r.Context())
	s.logAction(r, actor.Username, "employee_deleted", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// jpegMagic is the JPEG SOI marker: a minimal sanity check that the upload
// is actually a JPEG, since the destination filename is server-controlled
// (<employee_id>.jpg) and not taken from the upload itself.
var jpegMagic = []byte{0xFF, 0xD8, 0xFF}

func (s *Server) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := employees.ValidateEmployeeID(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, found, err := s.Employees.Store.Find(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not load employee: "+err.Error())
		return
	} else if !found {
		writeError(w, http.StatusNotFound, "no employee with that ID")
		return
	}

	if err := r.ParseMultipartForm(maxPhotoSize); err != nil {
		writeError(w, http.StatusBadRequest, "upload must be multipart/form-data under 5 MB: "+err.Error())
		return
	}
	file, _, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "a photo file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxPhotoSize+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the uploaded photo: "+err.Error())
		return
	}
	if len(data) > maxPhotoSize {
		writeError(w, http.StatusBadRequest, "photo is too large (max 5 MB)")
		return
	}
	if !bytes.HasPrefix(data, jpegMagic) {
		writeError(w, http.StatusBadRequest, "photo must be a JPEG file")
		return
	}

	zoom, err := formFloat(r, "zoom", 1, 2.5)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	x, err := formFloat(r, "x", 0, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	y, err := formFloat(r, "y", 0, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	filename := id + ".jpg"
	if err := storage.WriteFileAtomic(filepath.Join(s.Info.PhotosDir, filename), data); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save photo: "+err.Error())
		return
	}

	actor, _ := auth.UserFromContext(r.Context())
	e, err := s.Employees.Store.Update(id, func(e employees.Employee) (employees.Employee, error) {
		e.Photo = filename
		e.Crop = employees.Crop{Zoom: zoom, X: x, Y: y}
		e.UpdatedBy = actor.Username
		return e, nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "photo saved but employee record could not be updated: "+err.Error())
		return
	}
	s.logAction(r, actor.Username, "photo_uploaded", e.EmployeeID)
	writeJSON(w, http.StatusOK, e)
}

// handleGetPhoto serves an employee's uploaded photo. id is already
// restricted to employees.IDPattern (no path separators), so joining it
// straight onto PhotosDir cannot escape that directory.
func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, found, err := s.Employees.Store.Find(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load employee: "+err.Error())
		return
	}
	if !found || e.Photo == "" {
		writeError(w, http.StatusNotFound, "this employee has no photo")
		return
	}
	http.ServeFile(w, r, filepath.Join(s.Info.PhotosDir, e.Photo))
}

func formFloat(r *http.Request, field string, min, max float64) (float64, error) {
	v := r.FormValue(field)
	if v == "" {
		return 0, fmt.Errorf("%s is required", field)
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", field)
	}
	if f < min || f > max {
		return 0, fmt.Errorf("%s must be between %g and %g", field, min, max)
	}
	return f, nil
}
