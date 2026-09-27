// Package activity is the append-only audit log: sign-ins, user changes,
// client/employee/photo changes, imports and exports.
package activity

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is one line of data/activity.jsonl.
type Entry struct {
	TS       time.Time `json:"ts"`
	Username string    `json:"username"`
	IP       string    `json:"ip"`
	Host     string    `json:"host"`
	Action   string    `json:"action"`
	Target   string    `json:"target"`
}

// Log is the mutex-protected, append-only activity log file.
type Log struct {
	path string
	mu   sync.Mutex
}

// New returns a Log backed by the JSONL file at path.
func New(path string) *Log {
	return &Log{path: path}
}

// Write appends entry to the log, filling in TS with the current time if it
// is unset.
func (l *Log) Write(entry Entry) error {
	if entry.TS.IsZero() {
		entry.TS = time.Now()
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode activity entry: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", l.path, err)
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append to %s: %w", l.path, err)
	}
	return f.Sync()
}

// Filter narrows a Query; the zero value of each field means "no filter" on
// that field. Page is 1-based.
type Filter struct {
	Username string
	From     time.Time
	To       time.Time
	Search   string
	Page     int
	PageSize int
}

// Result is one page of a Query.
type Result struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
}

const defaultPageSize = 50

// Query loads every entry, applies filter, sorts newest-first and returns one
// page of matches plus the total match count (for pagination controls).
func (l *Log) Query(filter Filter) (Result, error) {
	l.mu.Lock()
	data, err := os.ReadFile(l.path)
	l.mu.Unlock()
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, nil
		}
		return Result{}, fmt.Errorf("read %s: %w", l.path, err)
	}

	var all []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip a corrupt line rather than fail the whole query
		}
		all = append(all, e)
	}

	search := strings.ToLower(filter.Search)
	filtered := make([]Entry, 0, len(all))
	for _, e := range all {
		if filter.Username != "" && !strings.EqualFold(e.Username, filter.Username) {
			continue
		}
		if !filter.From.IsZero() && e.TS.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.TS.After(filter.To) {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(e.Username + " " + e.IP + " " + e.Host + " " + e.Action + " " + e.Target)
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].TS.After(filtered[j].TS) })

	total := len(filtered)
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return Result{Entries: filtered[start:end], Total: total}, nil
}
