// Package domain (fitur monthlyplan) = Plan Monthly: target per tanggal dalam satu bulan, per mitra, deskripsi, dan nama.
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/plan"
)

// DayColumns = nama kolom tanggal di tabel core.monthly_plans (day_01..day_31; lama: T01..T31), juga nama isian JSON
// dan urutan kolom template Excel.
var DayColumns = func() []string {
	cols := make([]string, 31)
	for i := range cols {
		cols[i] = fmt.Sprintf("day_%02d", i+1)
	}
	return cols
}()

// MonthlyPlan = satu baris Plan Monthly (tabel core.monthly_plans; lama: Plan Month). Angka kosong disimpan 0.
// Total disimpan apa adanya dari input (kosong → jumlah tanggal).
type MonthlyPlan struct {
	ID          int64
	ProjectID   uuid.UUID
	Partner     string
	PlanYear    int
	PlanMonth   int
	Description string
	PlanName    string
	Uom         string
	Total       float64
	Days        [31]float64 // urutan = DayColumns
	UpdatedAt   time.Time
}

// DaysInMonth = jumlah hari pada bulan itu (Februari memperhitungkan tahun kabisat).
func DaysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// MarshalJSON: tanggal disajikan sebagai kolom datar ("day_01": 5075.38, ...), bukan larik.
func (p MonthlyPlan) MarshalJSON() ([]byte, error) {
	return plan.MarshalFlat(map[string]any{
		"id": p.ID, "project_id": p.ProjectID, "partner": p.Partner, "plan_year": p.PlanYear, "plan_month": p.PlanMonth,
		"description": p.Description, "plan_name": p.PlanName, "uom": p.Uom, "total": p.Total, "updated_at": p.UpdatedAt,
	}, DayColumns, p.Days[:])
}
