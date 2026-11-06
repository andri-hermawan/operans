// Package domain (fitur timesheet) = Time Sheet: hour meter (HM) awal/akhir tiap unit per tanggal operasional + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// TimeSheet = satu baris Time Sheet (tabel core.time_sheets; lama: Time Sheet). HM dalam jam.
// HmTotal disimpan apa adanya dari input (kosong → HmEnd − HmStart).
type TimeSheet struct {
	ID              int64      `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	OperationalDate string     `json:"operational_date"`
	ShiftID         uuid.UUID  `json:"shift_id"`
	ShiftCode       string     `json:"shift_code"`
	ShiftName       string     `json:"shift_name"`
	EquipmentCode   string     `json:"equipment_code"`
	EquipmentID     *uuid.UUID `json:"equipment_id"`
	OperatorNik     *string    `json:"operator_nik"`
	OperatorName    *string    `json:"operator_name"`
	HmStart         *float64   `json:"hm_start"`
	HmEnd           *float64   `json:"hm_end"`
	HmTotal         *float64   `json:"hm_total"`
	Remark          *string    `json:"remark"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
