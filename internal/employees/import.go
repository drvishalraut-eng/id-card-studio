package employees

// ImportRow is one row of an uploaded Employees sheet, already extracted
// from the .xlsx/.csv file client-side (by SheetJS) and sent as JSON.
type ImportRow struct {
	EmployeeID string `json:"employee_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Client     string `json:"client"`
	JoinDate   any    `json:"join_date"`
	PhotoNote  string `json:"photo_note"`
}

// SkippedRow reports why one import row was not applied.
type SkippedRow struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportResult summarizes a bulk import.
type ImportResult struct {
	Imported int          `json:"imported"`
	Skipped  []SkippedRow `json:"skipped"`
}

// Import upserts every valid row by employee_id, collecting a reason for
// each row it could not apply rather than failing the whole batch. Row
// numbers are 1-based, matching how a spreadsheet's rows are counted.
func (m *Manager) Import(rows []ImportRow, updatedBy string) (ImportResult, error) {
	result := ImportResult{Skipped: []SkippedRow{}}
	for i, row := range rows {
		_, err := m.Save(Input{
			EmployeeID: row.EmployeeID,
			Name:       row.Name,
			Role:       row.Role,
			Client:     row.Client,
			JoinDate:   row.JoinDate,
			PhotoNote:  row.PhotoNote,
		}, updatedBy)
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRow{Row: i + 1, Reason: err.Error()})
			continue
		}
		result.Imported++
	}
	return result, nil
}
