package domain

import (
	"time"

	"github.com/google/uuid"
)

// LiveState = kondisi TERAKHIR satu equipment (tabel iot.equipment_live_states).
// Disimpan di memori layanan dan di database; inilah yang ditampilkan peta live.
type LiveState struct {
	EquipmentID      uuid.UUID  `json:"equipment_id"`
	ProjectID        *uuid.UUID `json:"project_id"`
	DeviceID         uuid.UUID  `json:"device_id"`
	TelemetryEventID int64      `json:"telemetry_event_id"`
	EventTime        time.Time  `json:"event_time"`
	ReceivedAt       time.Time  `json:"received_at"`
	OperationalDate  *string    `json:"operational_date"` // YYYY-MM-DD (Fase C1)
	ShiftCode        *string    `json:"shift_code"`       // Fase C1

	// Posisi
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	SpeedKph   *float64 `json:"speed_kph"`
	HeadingDeg *int16   `json:"heading_deg"`

	// Status
	IsEngineOn      *bool    `json:"is_engine_on"`
	EquipmentStatus *string  `json:"equipment_status"`
	VesselStatus    *string  `json:"vessel_status"`
	VesselLoadTon   *float64 `json:"vessel_load_ton"`
	GSMSignalLevel  *int16   `json:"gsm_signal_level"`
	MileageM        *float64 `json:"mileage_m"`

	// Zona peta (Fase C2)
	ZoneID           *uuid.UUID `json:"zone_id"`
	IsInsideGeofence *bool      `json:"is_inside_geofence"`
	GeofenceFID      *int32     `json:"geofence_fid"`
	LocationCategory *string    `json:"location_category"`
	SegmentName      *string    `json:"segment_name"`

	// Acuan status muatan: zona terakhir di dalam peta (internal, tidak dikirim ke frontend)
	LastInsideFID          *int32  `json:"-"`
	LastInsideVesselStatus *string `json:"-"`

	// Keadaan pendeteksi kejadian (internal, kolom tracker JSON)
	Tracker Tracker `json:"-"`

	// Fuel (nilai terakhir yang dikirim)
	FuelLevelRaw     *float64 `json:"fuel_level_raw"`
	FuelVolumeLiter  *float64 `json:"fuel_volume_liter"`
	FuelPercent      *float64 `json:"fuel_percent"`
	FuelDeltaLiter   *float64 `json:"fuel_delta_liter"`
	FuelTemperatureC *float64 `json:"fuel_temperature_c"`

	// Operasional (operator & breakdown: Fase C1; alert_count: C4)
	OperatorName         *string `json:"operator_name"`
	IsBreakdown          *bool   `json:"is_breakdown"`
	BreakdownDescription *string `json:"breakdown_description"`
	AlertCount           int     `json:"alert_count"`

	UpdatedAt time.Time `json:"updated_at"`
}

// Owner = equipment, project, dan alat pemilik sebuah event (diambil dari iot.devices).
type Owner struct {
	EquipmentID uuid.UUID
	ProjectID   *uuid.UUID
	DeviceID    uuid.UUID
}

// NextLiveState menghitung kondisi terakhir yang baru dari kondisi sebelumnya (prev, boleh nil)
// dan event yang BARU SAJA tersimpan.
//
// changed=false bila event lebih lama daripada kondisi sekarang: data terlambat (alat sempat offline)
// tetap tersimpan di telemetry_events, tetapi tidak boleh menimpa posisi yang lebih baru di peta.
func NextLiveState(prev *LiveState, owner Owner, e *Event, telemetryID int64, receivedAt, now time.Time) (next LiveState, changed bool) {
	if prev != nil && e.EventTime.Before(prev.EventTime) {
		return *prev, false
	}
	if prev != nil {
		next = *prev // mulai dari kondisi lama: nilai yang tidak dikirim event ini tetap terbawa
	}

	next.EquipmentID = owner.EquipmentID
	next.ProjectID = owner.ProjectID
	next.DeviceID = owner.DeviceID
	next.TelemetryEventID = telemetryID
	next.EventTime = e.EventTime
	next.ReceivedAt = receivedAt
	next.UpdatedAt = now

	// Selalu mengikuti event terbaru (nil = memang tidak diketahui saat ini)
	next.SpeedKph = e.SpeedKph
	next.HeadingDeg = e.HeadingDeg
	next.IsEngineOn = e.IsEngineOn
	next.EquipmentStatus = e.EquipmentStatus
	next.GSMSignalLevel = e.GSMSignalLevel
	next.OperationalDate = e.OperationalDate
	next.ShiftCode = e.ShiftCode
	next.OperatorName = e.OperatorName
	next.IsBreakdown = e.IsBreakdown
	next.BreakdownDescription = e.BreakdownDescription
	next.VesselStatus = e.VesselStatus // ApplyVessel sudah membawa nilai lama bila zona tidak diketahui
	next.LastInsideFID, next.LastInsideVesselStatus = e.LastInsideFID, e.LastInsideVesselStatus
	if e.AlertCount != nil {
		next.AlertCount = *e.AlertCount
	}
	if e.Tracker != nil {
		next.Tracker = *e.Tracker
	}
	if e.GeofenceFID != nil { // zona diketahui (ada posisi & peta); bila tidak, zona terakhir tetap tampil
		next.ZoneID, next.IsInsideGeofence, next.GeofenceFID = e.ZoneID, e.IsInsideGeofence, e.GeofenceFID
		next.LocationCategory, next.SegmentName = e.LocationCategory, e.SegmentName
	}

	// Dipertahankan bila event ini tidak membawanya
	if e.Latitude != nil && e.Longitude != nil { // posisi terakhir yang diketahui tetap tampil di peta
		next.Latitude, next.Longitude = e.Latitude, e.Longitude
	}
	if e.VesselLoadTon != nil {
		next.VesselLoadTon = e.VesselLoadTon
	}
	if e.MileageM != nil {
		next.MileageM = e.MileageM
	}
	if e.FuelLevelRaw != nil {
		next.FuelLevelRaw = e.FuelLevelRaw
	}
	if e.FuelVolumeLiter != nil {
		next.FuelVolumeLiter = e.FuelVolumeLiter
	}
	if e.FuelPercent != nil {
		next.FuelPercent = e.FuelPercent
	}
	if e.FuelDeltaLiter != nil {
		next.FuelDeltaLiter = e.FuelDeltaLiter
	}
	if e.FuelTemperatureC != nil {
		next.FuelTemperatureC = e.FuelTemperatureC
	}
	return next, true
}
