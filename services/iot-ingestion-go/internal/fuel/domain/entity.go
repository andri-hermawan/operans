// Package domain = bentuk data fitur fuel (kejadian isi/turun BBM; baris iot.fuel_events).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// FuelEventRow = kejadian isi/turun BBM (penanda di Fuel History).
type FuelEventRow struct {
	ID          int64     `json:"id"`
	EquipmentID uuid.UUID `json:"equipment_id"`
	EventType   string    `json:"event_type"`
	EventTime   time.Time `json:"event_time"`
	DeltaLiter  float64   `json:"fuel_delta_liter"`
	Volume      *float64  `json:"fuel_volume_liter"`
	Description *string   `json:"description"`
	AlertID     *int64    `json:"alert_id"`
}
