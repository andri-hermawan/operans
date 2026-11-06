// Package domain (fitur operator) = operator yang menjalankan unit per tanggal operasional + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// OperatorAssignment = operator yang mengoperasikan unit pada satu tanggal operasional + shift
// (tabel core.operator_assignments; lama: daily_setting_operator).
type OperatorAssignment struct {
	ID              int64      `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	OperationalDate string     `json:"operational_date"` // YYYY-MM-DD
	ShiftID         uuid.UUID  `json:"shift_id"`
	ShiftCode       string     `json:"shift_code"`
	ShiftName       string     `json:"shift_name"`
	EquipmentCode   string     `json:"equipment_code"`
	EquipmentID     *uuid.UUID `json:"equipment_id"` // terisi bila unit ada di master equipment
	OperatorName    string     `json:"operator_name"`
	Description     *string    `json:"description"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
