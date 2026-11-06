// Package domain = bentuk data fitur telemetry (hasil baca iot.telemetry_events untuk dashboard:
// latest/history, titik riwayat & heatmap, speed per segmen, ringkasan aktivitas).
package domain

import (
	"time"

	"github.com/google/uuid"

	ingestdomain "simha/iot-ingestion/internal/ingest/domain"
)

// TelemetryRow = satu baris hasil baca. Data teknis (listrik, accelerometer, ...) belum ikut;
// tambahkan kolomnya di telemetryCols + scanTelemetry bila dashboard membutuhkannya.
type TelemetryRow struct {
	ID          int64      `json:"id"`
	EventID     string     `json:"event_id"`
	DeviceID    uuid.UUID  `json:"device_id"`
	EquipmentID *uuid.UUID `json:"equipment_id"`
	ProjectID   *uuid.UUID `json:"project_id"`
	EventTime   time.Time  `json:"event_time"`
	CreatedAt   time.Time  `json:"created_at"` // waktu diterima server
	ingestdomain.Position
	ingestdomain.Engine
	ingestdomain.Fuel
	EquipmentStatus  *string  `json:"equipment_status"`
	VesselStatus     *string  `json:"vessel_status"`
	VesselLoadTon    *float64 `json:"vessel_load_ton"`
	LocationCategory *string  `json:"location_category"`
	SegmentName      *string  `json:"segment_name"`
	OperationalDate  *string  `json:"operational_date"`
	ShiftCode        *string  `json:"shift_code"`
	OperatorName     *string  `json:"operator_name"`
	IsBreakdown      *bool    `json:"is_breakdown"`
	IsInsideGeofence *bool    `json:"is_inside_geofence"`
	GeofenceFID      *int32   `json:"geofence_fid"`
}

// Point = satu titik telemetry ringkas (peta riwayat, playback, heatmap, grafik).
type Point struct {
	ID              int64     `json:"id"`
	EquipmentID     uuid.UUID `json:"equipment_id"`
	EventTime       time.Time `json:"event_time"`
	Latitude        *float64  `json:"latitude"`
	Longitude       *float64  `json:"longitude"`
	SpeedKph        *float64  `json:"speed_kph"`
	HeadingDeg      *int16    `json:"heading_deg"`
	IsEngineOn      *bool     `json:"is_engine_on"`
	EquipmentStatus *string   `json:"equipment_status"`
	VesselStatus    *string   `json:"vessel_status"`
	FuelVolume      *float64  `json:"fuel_volume_liter"`
	FuelPercent     *float64  `json:"fuel_percent"`
	FuelDelta       *float64  `json:"fuel_delta_liter"`
	FuelTemperature *float64  `json:"fuel_temperature_c"`
	SegmentName     *string   `json:"segment_name"`
	Category        *string   `json:"location_category"`
	IsInside        *bool     `json:"is_inside_geofence"`
	OperatorName    *string   `json:"operator_name"`
	IsBreakdown     *bool     `json:"is_breakdown"`
	ShiftCode       *string   `json:"shift_code"`
	OperationalDate *string   `json:"operational_date"`
}

// SegmentSpeed = rata-rata speed per segmen jalan, dipisah kosong/bermuatan (Speed Heatmap, sama dengan lama:
// hanya titik yang bergerak).
type SegmentSpeed struct {
	SegmentName string   `json:"segment_name"`
	AvgEmpty    *float64 `json:"avg_speed_empty"`
	AvgLoaded   *float64 `json:"avg_speed_loaded"`
	AvgAll      *float64 `json:"avg_speed"`
	Points      int      `json:"points"`
}

// Activity = ringkasan aktivitas satu unit dalam satu tanggal/shift (panel Live Tracking).
// Rumus sama dengan sistem lama (equipment-logs.repository getActivitySummary):
//   - jam running/idle = jumlah selisih waktu ke data berikutnya menurut status data itu (kosong/bermuatan saja)
//   - speed rata-rata/maks = data RUNNING dengan speed > 0
//   - mileage = (maks − min) mileage, km
//   - fuel turun/naik = jumlah selisih liter per data (lama: tabel fuels per data); awal/akhir = liter pertama/terakhir
//   - burn ratio = km ÷ liter turun
type Activity struct {
	EquipmentID      uuid.UUID `json:"equipment_id"`
	RunningHours     float64   `json:"running_hours"`
	RunningEmpty     float64   `json:"running_empty_hours"`
	RunningLoaded    float64   `json:"running_loaded_hours"`
	IdleHours        float64   `json:"idle_hours"`
	IdleEmpty        float64   `json:"idle_empty_hours"`
	IdleLoaded       float64   `json:"idle_loaded_hours"`
	AvgSpeed         float64   `json:"avg_speed"`
	AvgSpeedEmpty    float64   `json:"avg_speed_empty"`
	AvgSpeedLoaded   float64   `json:"avg_speed_loaded"`
	MaxSpeed         float64   `json:"max_speed"`
	MaxSpeedEmpty    float64   `json:"max_speed_empty"`
	MaxSpeedLoaded   float64   `json:"max_speed_loaded"`
	MileageKm        float64   `json:"mileage_km"`
	FuelStart        *float64  `json:"fuel_start_liter"`
	FuelRemaining    *float64  `json:"fuel_remaining_liter"`
	FuelIncrease     float64   `json:"fuel_increase_liter"`
	FuelDecrease     float64   `json:"fuel_decrease_liter"`
	FuelBurnKmPerLtr float64   `json:"fuel_burn_km_per_liter"`
	Points           int       `json:"points"`
}
