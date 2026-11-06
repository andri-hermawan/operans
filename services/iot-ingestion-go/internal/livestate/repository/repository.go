// Package repository = SQL fitur livestate: tabel iot.equipment_live_states (kondisi terakhir per equipment).
// Dibaca SEKALI saat start (LoadLiveStates), lalu hanya ditulis (UpsertLiveState) oleh service.Cache.
package repository

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/shared/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Daftar kolom TETAP; urutannya HARUS sama dengan urutan Scan di LoadLiveStates.
const liveStateCols = `
    equipment_id, project_id, device_id, telemetry_event_id, event_time, received_at,
    operational_date::text, shift_code,
    ST_Y(location), ST_X(location), speed_kph, heading_deg,
    is_engine_on, equipment_status, vessel_status, vessel_load_ton, gsm_signal_level, mileage_m,
    is_inside_geofence, geofence_fid, location_category, segment_name,
    fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
    operator_name, is_breakdown, breakdown_description, alert_count, updated_at,
    zone_id, last_inside_fid, last_inside_vessel_status, tracker`

// LoadLiveStates membaca SEMUA kondisi terakhir (satu baris per equipment).
// Dipanggil sekali saat layanan start untuk mengisi cache di memori.
func (r *Repository) LoadLiveStates(ctx context.Context) ([]domain.LiveState, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+liveStateCols+` FROM iot.equipment_live_states`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.LiveState{}
	for rows.Next() {
		var s domain.LiveState
		if err := rows.Scan(
			&s.EquipmentID, &s.ProjectID, &s.DeviceID, &s.TelemetryEventID, &s.EventTime, &s.ReceivedAt,
			&s.OperationalDate, &s.ShiftCode,
			&s.Latitude, &s.Longitude, &s.SpeedKph, &s.HeadingDeg,
			&s.IsEngineOn, &s.EquipmentStatus, &s.VesselStatus, &s.VesselLoadTon, &s.GSMSignalLevel, &s.MileageM,
			&s.IsInsideGeofence, &s.GeofenceFID, &s.LocationCategory, &s.SegmentName,
			&s.FuelLevelRaw, &s.FuelVolumeLiter, &s.FuelPercent, &s.FuelDeltaLiter, &s.FuelTemperatureC,
			&s.OperatorName, &s.IsBreakdown, &s.BreakdownDescription, &s.AlertCount, &s.UpdatedAt,
			&s.ZoneID, &s.LastInsideFID, &s.LastInsideVesselStatus, &s.Tracker,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// upsertLiveStateSQL: insert bila equipment belum punya baris, timpa bila sudah ada.
// WHERE di bagian UPDATE = pengaman kedua: data yang lebih lama tidak pernah menimpa yang lebih baru.
const upsertLiveStateSQL = `
INSERT INTO iot.equipment_live_states (
    equipment_id, project_id, device_id, telemetry_event_id, event_time, received_at,
    operational_date, shift_code,
    location, speed_kph, heading_deg,
    is_engine_on, equipment_status, vessel_status, vessel_load_ton, gsm_signal_level, mileage_m,
    is_inside_geofence, geofence_fid, location_category, segment_name,
    fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
    operator_name, is_breakdown, breakdown_description, alert_count, updated_at,
    zone_id, last_inside_fid, last_inside_vessel_status, tracker
) VALUES (
    @equipment_id, @project_id, @device_id, @telemetry_event_id, @event_time, @received_at,
    @operational_date::text::date, @shift_code,
    ST_SetSRID(ST_MakePoint(@longitude, @latitude), 4326), @speed_kph, @heading_deg,
    @is_engine_on, @equipment_status, @vessel_status, @vessel_load_ton, @gsm_signal_level, @mileage_m,
    @is_inside_geofence, @geofence_fid, @location_category, @segment_name,
    @fuel_level_raw, @fuel_volume_liter, @fuel_percent, @fuel_delta_liter, @fuel_temperature_c,
    @operator_name, @is_breakdown, @breakdown_description, @alert_count, @updated_at,
    @zone_id, @last_inside_fid, @last_inside_vessel_status, @tracker
)
ON CONFLICT (equipment_id) DO UPDATE SET
    project_id = EXCLUDED.project_id, device_id = EXCLUDED.device_id,
    telemetry_event_id = EXCLUDED.telemetry_event_id, event_time = EXCLUDED.event_time, received_at = EXCLUDED.received_at,
    operational_date = EXCLUDED.operational_date, shift_code = EXCLUDED.shift_code,
    location = EXCLUDED.location, speed_kph = EXCLUDED.speed_kph, heading_deg = EXCLUDED.heading_deg,
    is_engine_on = EXCLUDED.is_engine_on, equipment_status = EXCLUDED.equipment_status,
    vessel_status = EXCLUDED.vessel_status, vessel_load_ton = EXCLUDED.vessel_load_ton,
    gsm_signal_level = EXCLUDED.gsm_signal_level, mileage_m = EXCLUDED.mileage_m,
    is_inside_geofence = EXCLUDED.is_inside_geofence, geofence_fid = EXCLUDED.geofence_fid,
    location_category = EXCLUDED.location_category, segment_name = EXCLUDED.segment_name,
    fuel_level_raw = EXCLUDED.fuel_level_raw, fuel_volume_liter = EXCLUDED.fuel_volume_liter,
    fuel_percent = EXCLUDED.fuel_percent, fuel_delta_liter = EXCLUDED.fuel_delta_liter,
    fuel_temperature_c = EXCLUDED.fuel_temperature_c,
    operator_name = EXCLUDED.operator_name, is_breakdown = EXCLUDED.is_breakdown,
    breakdown_description = EXCLUDED.breakdown_description, alert_count = EXCLUDED.alert_count,
    updated_at = EXCLUDED.updated_at, zone_id = EXCLUDED.zone_id,
    last_inside_fid = EXCLUDED.last_inside_fid, last_inside_vessel_status = EXCLUDED.last_inside_vessel_status,
    tracker = EXCLUDED.tracker
WHERE iot.equipment_live_states.event_time <= EXCLUDED.event_time`

// UpsertLiveState menyimpan kondisi terakhir satu equipment. Bila koneksi database putus di tengah jalan
// (mis. diputus dari sisi server), dicoba SEKALI lagi dengan koneksi baru: aman karena upsert ini idempoten
// (data lama tidak pernah menimpa yang lebih baru).
func (r *Repository) UpsertLiveState(ctx context.Context, s domain.LiveState) error {
	args := pgx.NamedArgs{
		"equipment_id": s.EquipmentID, "project_id": s.ProjectID, "device_id": s.DeviceID,
		"telemetry_event_id": s.TelemetryEventID, "event_time": s.EventTime.UTC(), "received_at": s.ReceivedAt.UTC(),
		"operational_date": s.OperationalDate, "shift_code": s.ShiftCode,
		"latitude": s.Latitude, "longitude": s.Longitude, "speed_kph": s.SpeedKph, "heading_deg": s.HeadingDeg,
		"is_engine_on": s.IsEngineOn, "equipment_status": s.EquipmentStatus, "vessel_status": s.VesselStatus,
		"vessel_load_ton": s.VesselLoadTon, "gsm_signal_level": s.GSMSignalLevel, "mileage_m": s.MileageM,
		"is_inside_geofence": s.IsInsideGeofence, "geofence_fid": s.GeofenceFID,
		"location_category": s.LocationCategory, "segment_name": s.SegmentName,
		"fuel_level_raw": s.FuelLevelRaw, "fuel_volume_liter": s.FuelVolumeLiter, "fuel_percent": s.FuelPercent,
		"fuel_delta_liter": s.FuelDeltaLiter, "fuel_temperature_c": s.FuelTemperatureC,
		"operator_name": s.OperatorName, "is_breakdown": s.IsBreakdown, "breakdown_description": s.BreakdownDescription,
		"alert_count": s.AlertCount, "updated_at": s.UpdatedAt.UTC(),
		"zone_id": s.ZoneID, "last_inside_fid": s.LastInsideFID, "last_inside_vessel_status": s.LastInsideVesselStatus,
		"tracker": s.Tracker,
	}
	_, err := r.pool.Exec(ctx, upsertLiveStateSQL, args)
	if err != nil && database.IsConnectionError(err) && ctx.Err() == nil {
		slog.Warn("koneksi database putus saat simpan kondisi terakhir, dicoba ulang", "equipment_id", s.EquipmentID, "err", err)
		_, err = r.pool.Exec(ctx, upsertLiveStateSQL, args)
	}
	return err
}
