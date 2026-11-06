// Package domain (fitur breakdown) = catatan unit rusak/berhenti per tanggal operasional + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status breakdown (ADR 0004/0005): kejadian nyata | berlanjut sampai akhir shift | selesai.
const (
	BreakdownStatusBreakdown = "breakdown"
	BreakdownStatusContinue  = "continue"
	BreakdownStatusReady     = "ready"
)

// EquipmentBreakdown = catatan unit rusak/berhenti pada satu tanggal operasional + shift
// (tabel core.equipment_breakdowns; lama: breakdown_status).
type EquipmentBreakdown struct {
	ID                int64      `json:"id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	OperationalDate   string     `json:"operational_date"`
	ShiftID           uuid.UUID  `json:"shift_id"`
	ShiftCode         string     `json:"shift_code"`
	ShiftName         string     `json:"shift_name"`
	EquipmentCode     string     `json:"equipment_code"`
	EquipmentID       *uuid.UUID `json:"equipment_id"`
	Status            string     `json:"status"`
	CategoryCode      string     `json:"category_code"`
	CategoryLabel     string     `json:"category_label"`
	StartedAt         *time.Time `json:"started_at"`
	EndedAt           *time.Time `json:"ended_at"`
	DurationHours     *float64   `json:"duration_hours"`
	RepairStatusCode  string     `json:"repair_status_code"`
	RepairStatusLabel string     `json:"repair_status_label"`
	Description       *string    `json:"description"`
	Location          *string    `json:"location"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
