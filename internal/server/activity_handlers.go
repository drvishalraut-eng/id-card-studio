package server

import (
	"net/http"
	"strconv"
	"time"

	"idcardstudio/internal/activity"
)

func (s *Server) registerActivityRoutes(mux *http.ServeMux) {
	mux.Handle("GET /activity", s.Auth.RequireAuth(http.HandlerFunc(s.handleListActivity)))
}

// dateLayout matches the plain YYYY-MM-DD dates used by the Activity page's
// date-range filter.
const dateLayout = "2006-01-02"

func (s *Server) handleListActivity(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := activity.Filter{
		Username: q.Get("username"),
		Search:   q.Get("search"),
	}

	if v := q.Get("from"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "from must be a date in YYYY-MM-DD format")
			return
		}
		filter.From = t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "to must be a date in YYYY-MM-DD format")
			return
		}
		// Include the whole day.
		filter.To = t.Add(24*time.Hour - time.Nanosecond)
	}
	if v := q.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "page must be a positive integer")
			return
		}
		filter.Page = n
	}
	if v := q.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "page_size must be between 1 and 200")
			return
		}
		filter.PageSize = n
	}

	result, err := s.Activity.Query(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load activity log: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
