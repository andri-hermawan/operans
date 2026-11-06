// Package domain (fitur importexcel) = hasil pratinjau / simpan upload Excel.
package domain

import "simha/core-api/internal/shared/dailyinput"

// ImportResult = hasil pratinjau / simpan impor (ADR 0007).
type ImportResult struct {
	Type          string                `json:"type"`
	DryRun        bool                  `json:"dry_run"`
	Saved         bool                  `json:"saved"`
	TotalRows     int                   `json:"total_rows"`
	ValidRows     int                   `json:"valid_rows"`
	InvalidRows   int                   `json:"invalid_rows"`
	DuplicateRows int                   `json:"duplicate_rows"` // baris yang ditimpa baris lain dengan kunci sama
	ToCreate      int                   `json:"to_create"`
	ToUpdate      int                   `json:"to_update"`
	Errors        []dailyinput.RowIssue `json:"errors"`
	Warnings      []dailyinput.RowIssue `json:"warnings"`
}
