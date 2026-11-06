// Package domain (fitur yearlyplan) = Plan Yearly: target per bulan dalam satu tahun, per mitra, lokasi, deskripsi, dan nama.
package domain

import (
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/plan"
)

// MonthColumns = nama kolom bulan di tabel core.yearly_plans (Jan..Des), juga nama isian JSON dan urutan kolom template Excel.
var MonthColumns = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

// YearlyPlan = satu baris Plan Yearly (tabel core.yearly_plans; lama: Plan Yearly). Angka kosong disimpan 0.
// Total disimpan apa adanya dari input (kosong → jumlah bulan).
type YearlyPlan struct {
	ID          int64
	ProjectID   uuid.UUID
	Partner     string
	PlanYear    int
	Location    string
	Description string
	PlanName    string
	Total       float64
	Months      [12]float64 // urutan = MonthColumns
	UpdatedAt   time.Time
}

// MarshalJSON: bulan disajikan sebagai kolom datar ("jan": 180055.41, ...), bukan larik.
func (p YearlyPlan) MarshalJSON() ([]byte, error) {
	return plan.MarshalFlat(map[string]any{
		"id": p.ID, "project_id": p.ProjectID, "partner": p.Partner, "plan_year": p.PlanYear, "location": p.Location,
		"description": p.Description, "plan_name": p.PlanName, "total": p.Total, "updated_at": p.UpdatedAt,
	}, MonthColumns, p.Months[:])
}
