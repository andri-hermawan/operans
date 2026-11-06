package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Kode jenis alert (sama dengan core.alert_types & iot.alerts.alert_type).
const (
	AlertOffTrack              = "off_track"
	AlertOverspeed             = "overspeed"
	AlertUnderspeed            = "underspeed"
	AlertFuelDecreaseEngineOn  = "fuel_decrease_engine_on"
	AlertFuelDecreaseEngineOff = "fuel_decrease_engine_off"
)

// AlertRule = ambang efektif satu jenis alert untuk project + zona (core.effective_alert_rules).
type AlertRule struct {
	DurationMinutes *float64
	SpeedLimitKph   *float64
	FuelDropLiters  *float64
	Recipients      []uuid.UUID // penerima WhatsApp (C5)
}

// AlertRules = ambang efektif per jenis alert.
type AlertRules map[string]AlertRule

// DefaultAlertRules = bawaan bila data induk belum termuat (= aturan global sistem lama, ADR 0008).
var DefaultAlertRules = AlertRules{
	AlertOffTrack:              {DurationMinutes: f(3)},
	AlertOverspeed:             {DurationMinutes: f(1), SpeedLimitKph: f(50)},
	AlertUnderspeed:            {DurationMinutes: f(2), SpeedLimitKph: f(10)},
	AlertFuelDecreaseEngineOn:  {DurationMinutes: f(2), FuelDropLiters: f(5)},
	AlertFuelDecreaseEngineOff: {FuelDropLiters: f(5)},
}

func f(v float64) *float64 { return &v }

// num = nilai ambang (0 bila kosong).
func num(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// ErrAlertAlreadyOpen = alert jenis ini untuk unit ini masih terbuka (index unik uq_alerts_one_open).
var ErrAlertAlreadyOpen = errors.New("alert masih terbuka")

// Alert = satu baris iot.alerts (kejadian dengan awal & akhir).
type Alert struct {
	ID               int64       `json:"id"`
	ProjectID        *uuid.UUID  `json:"project_id"`
	EquipmentID      uuid.UUID   `json:"equipment_id"`
	DeviceID         uuid.UUID   `json:"device_id"`
	TelemetryEventID int64       `json:"telemetry_event_id"`
	AlertType        string      `json:"alert_type"`
	StartedAt        time.Time   `json:"started_at"`
	EndedAt          *time.Time  `json:"ended_at"`
	OperationalDate  *string     `json:"operational_date"`
	ShiftCode        *string     `json:"shift_code"`
	ThresholdValue   *float64    `json:"threshold_value"`
	Snapshot                     // kondisi saat kejadian
	OperatorName     *string     `json:"operator_name"`
	Recipients       []uuid.UUID `json:"-"`
}

// FuelEvent = satu baris iot.fuel_events (isi / turun BBM).
type FuelEvent struct {
	ID               int64      `json:"id"`
	ProjectID        *uuid.UUID `json:"project_id"`
	EquipmentID      uuid.UUID  `json:"equipment_id"`
	DeviceID         uuid.UUID  `json:"device_id"`
	TelemetryEventID int64      `json:"telemetry_event_id"`
	EventType        string     `json:"event_type"` // increase | decrease
	EventTime        time.Time  `json:"event_time"`
	OperationalDate  *string    `json:"operational_date"`
	ShiftCode        *string    `json:"shift_code"`
	FuelDeltaLiter   float64    `json:"fuel_delta_liter"` // negatif = turun
	Description      string     `json:"description"`
	AlertID          *int64     `json:"alert_id"`
	Snapshot
}

// Snapshot = kondisi unit pada data yang memicu kejadian (diisi dari event).
type Snapshot struct {
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	SpeedKph         *float64 `json:"speed_kph"`
	IsEngineOn       *bool    `json:"is_engine_on"`
	EquipmentStatus  *string  `json:"equipment_status"`
	VesselStatus     *string  `json:"vessel_status"`
	MileageM         *float64 `json:"mileage_m"`
	FuelLevelRaw     *float64 `json:"fuel_level_raw"`
	FuelVolumeLiter  *float64 `json:"fuel_volume_liter"`
	FuelPercent      *float64 `json:"fuel_percent"`
	FuelDeltaLiter   *float64 `json:"fuel_delta_liter"`
	FuelTemperatureC *float64 `json:"fuel_temperature_c"`
	IsInsideGeofence *bool    `json:"is_inside_geofence"`
	GeofenceFID      *int32   `json:"geofence_fid"`
	LocationCategory *string  `json:"location_category"`
	SegmentName      *string  `json:"segment_name"`
}

// SnapshotOf mengambil kondisi unit dari event.
func SnapshotOf(e *Event) Snapshot {
	return Snapshot{Latitude: e.Latitude, Longitude: e.Longitude, SpeedKph: e.SpeedKph, IsEngineOn: e.IsEngineOn,
		EquipmentStatus: e.EquipmentStatus, VesselStatus: e.VesselStatus, MileageM: e.MileageM,
		FuelLevelRaw: e.FuelLevelRaw, FuelVolumeLiter: e.FuelVolumeLiter, FuelPercent: e.FuelPercent,
		FuelDeltaLiter: e.FuelDeltaLiter, FuelTemperatureC: e.FuelTemperatureC, IsInsideGeofence: e.IsInsideGeofence,
		GeofenceFID: e.GeofenceFID, LocationCategory: e.LocationCategory, SegmentName: e.SegmentName}
}
