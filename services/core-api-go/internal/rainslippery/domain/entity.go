// Package domain (fitur rainslippery) = Rain & Slippery: waktu hujan, tunggu licin, dan licin per lokasi + shift.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// RainSlipperyLog = satu baris Rain & Slippery (tabel core.rain_slippery_logs).
// Tiap pasangan jam (hujan, tunggu licin, licin) boleh kosong; total jam kosong → dihitung dari jam.
type RainSlipperyLog struct {
	ID                  int64      `json:"id"`
	ProjectID           uuid.UUID  `json:"project_id"`
	OperationalDate     string     `json:"operational_date"`
	ShiftID             uuid.UUID  `json:"shift_id"`
	ShiftCode           string     `json:"shift_code"`
	ShiftName           string     `json:"shift_name"`
	Location            *string    `json:"location"`
	RainStartAt         *time.Time `json:"rain_start_at"`
	RainStopAt          *time.Time `json:"rain_stop_at"`
	RainHours           *float64   `json:"rain_hours"`
	WaitSlipperyStartAt *time.Time `json:"wait_slippery_start_at"`
	WaitSlipperyStopAt  *time.Time `json:"wait_slippery_stop_at"`
	WaitSlipperyHours   *float64   `json:"wait_slippery_hours"`
	SlipperyStartAt     *time.Time `json:"slippery_start_at"`
	SlipperyStopAt      *time.Time `json:"slippery_stop_at"`
	SlipperyHours       *float64   `json:"slippery_hours"`
	RainfallMm          *float64   `json:"rainfall_mm"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
