// Package domain (fitur workhour) = WH & Notification: kejadian jam kerja unit (standby, dll) per tanggal + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// WorkHourNotification = satu baris WH & Notification (tabel core.work_hour_notifications).
// TotalHours kosong pada input → dihitung dari selisih jam selesai − mulai.
type WorkHourNotification struct {
	ID              int64      `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	OperationalDate string     `json:"operational_date"`
	ShiftID         uuid.UUID  `json:"shift_id"`
	ShiftCode       string     `json:"shift_code"`
	ShiftName       string     `json:"shift_name"`
	EquipmentCode   string     `json:"equipment_code"`
	EquipmentID     *uuid.UUID `json:"equipment_id"`
	Category        string     `json:"category"`
	SubCategory     *string    `json:"sub_category"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	TotalHours      *float64   `json:"total_hours"`
	Dedicated       *string    `json:"dedicated"`
	Remark          *string    `json:"remark"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
