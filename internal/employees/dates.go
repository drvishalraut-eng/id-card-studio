package employees

import (
	"fmt"
	"strings"
	"time"
)

// excelEpoch is Excel's day-zero: 30 Dec 1899. Using this (rather than the
// nominal 1 Jan 1900) reproduces Excel's date serials correctly, including
// its well-known — but universally replicated — 1900 leap-year bug.
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// ParseDateValue normalizes a join_date cell to "YYYY-MM-DD". v may be a
// string in "YYYY-MM-DD" or "DD-MM-YYYY" format, or a float64 Excel date
// serial (as produced by decoding JSON row data extracted client-side by
// SheetJS).
func ParseDateValue(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "", fmt.Errorf("join_date is required")
	case float64:
		return excelSerialToDate(val)
	case int:
		return excelSerialToDate(float64(val))
	case string:
		return parseDateString(val)
	default:
		return "", fmt.Errorf("join_date has an unsupported type (%T)", v)
	}
}

func parseDateString(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("join_date is required")
	}
	for _, layout := range []string{"2006-01-02", "02-01-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("join_date %q is not a recognized date (use YYYY-MM-DD or DD-MM-YYYY)", s)
}

func excelSerialToDate(serial float64) (string, error) {
	if serial <= 0 {
		return "", fmt.Errorf("join_date %.0f is not a valid Excel date serial", serial)
	}
	return excelEpoch.AddDate(0, 0, int(serial)).Format("2006-01-02"), nil
}
