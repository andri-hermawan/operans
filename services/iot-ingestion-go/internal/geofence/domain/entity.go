// Package domain = bentuk data fitur geofence (kejadian masuk/keluar area; baris iot.geofence_events).
package domain

import (
	"time"

	"github.com/google/uuid"
)

// GeofenceRow = kejadian masuk/keluar area (Geofencing).
type GeofenceRow struct {
	ID          int64     `json:"id"`
	EquipmentID uuid.UUID `json:"equipment_id"`
	EventType   string    `json:"event_type"` // enter | exit
	EventTime   time.Time `json:"event_time"`
	ShiftCode   *string   `json:"shift_code"`
	SegmentName *string   `json:"segment_name"`
	Category    *string   `json:"location_category"`
	Latitude    *float64  `json:"latitude"`
	Longitude   *float64  `json:"longitude"`
	SpeedKph    *float64  `json:"speed_kph"`
	Vessel      *string   `json:"vessel_status"`
}
