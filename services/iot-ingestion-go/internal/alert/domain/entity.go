// Package domain = bentuk data fitur alert (daftar alert & peta sebaran; baris iot.alerts).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AlertRow = alert untuk daftar & peta sebaran.
type AlertRow struct {
	ID              int64      `json:"id"`
	EquipmentID     uuid.UUID  `json:"equipment_id"`
	AlertType       string     `json:"alert_type"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
	OperationalDate *string    `json:"operational_date"`
	ShiftCode       *string    `json:"shift_code"`
	Threshold       *float64   `json:"threshold_value"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
	SpeedKph        *float64   `json:"speed_kph"`
	IsEngineOn      *bool      `json:"is_engine_on"`
	VesselStatus    *string    `json:"vessel_status"`
	FuelVolume      *float64   `json:"fuel_volume_liter"`
	FuelPercent     *float64   `json:"fuel_percent"`
	FuelDelta       *float64   `json:"fuel_delta_liter"`
	FuelTemperature *float64   `json:"fuel_temperature_c"`
	SegmentName     *string    `json:"segment_name"`
	Category        *string    `json:"location_category"`
	OperatorName    *string    `json:"operator_name"`
}
