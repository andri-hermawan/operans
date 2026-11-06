// Package domain (fitur reference) = daftar pilihan (kategori breakdown, status perbaikan, …).
package domain

// ReferenceValue = satu pilihan dalam daftar (kategori breakdown, status perbaikan, …).
type ReferenceValue struct {
	GroupCode string `json:"group_code"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
}

// Kelompok daftar pilihan di core.reference_values.
const (
	RefBreakdownCategory = "breakdown_category"
	RefRepairStatus      = "repair_status"
)
