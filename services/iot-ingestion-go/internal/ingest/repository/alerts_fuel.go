package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"simha/iot-ingestion/internal/ingest/domain"
)

// snapshotArgs = kolom kondisi unit yang sama di iot.alerts & iot.fuel_events.
func snapshotArgs(a pgx.NamedArgs, s domain.Snapshot) pgx.NamedArgs {
	a["latitude"], a["longitude"], a["speed_kph"], a["is_engine_on"] = s.Latitude, s.Longitude, s.SpeedKph, s.IsEngineOn
	a["equipment_status"], a["vessel_status"], a["mileage_m"] = s.EquipmentStatus, s.VesselStatus, s.MileageM
	a["fuel_level_raw"], a["fuel_volume_liter"], a["fuel_percent"] = s.FuelLevelRaw, s.FuelVolumeLiter, s.FuelPercent
	a["fuel_temperature_c"], a["is_inside_geofence"], a["geofence_fid"] = s.FuelTemperatureC, s.IsInsideGeofence, s.GeofenceFID
	a["location_category"], a["segment_name"] = s.LocationCategory, s.SegmentName
	return a
}

// InsertAlert menyimpan satu alert dan mengisi a.ID. Alert terbuka (ended_at kosong) yang sudah ada untuk
// unit + jenis yang sama → domain.ErrAlertAlreadyOpen (tidak dobel, mis. setelah keadaan pendeteksi hilang).
func (r *Repository) InsertAlert(ctx context.Context, a *domain.Alert) error {
	args := snapshotArgs(pgx.NamedArgs{
		"project_id": a.ProjectID, "equipment_id": a.EquipmentID, "device_id": a.DeviceID,
		"telemetry_event_id": a.TelemetryEventID, "alert_type": a.AlertType, "started_at": a.StartedAt.UTC(),
		"ended_at": a.EndedAt, "operational_date": a.OperationalDate, "shift_code": a.ShiftCode,
		"threshold_value": a.ThresholdValue, "fuel_delta_liter": a.FuelDeltaLiter, "operator_name": a.OperatorName,
	}, a.Snapshot)
	err := r.pool.QueryRow(ctx, `
		INSERT INTO iot.alerts (
		    project_id, equipment_id, device_id, telemetry_event_id, alert_type, started_at, ended_at,
		    operational_date, shift_code, threshold_value,
		    location, speed_kph, is_engine_on, equipment_status, vessel_status, mileage_m,
		    fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
		    is_inside_geofence, geofence_fid, location_category, segment_name, operator_name
		) VALUES (
		    @project_id, @equipment_id, @device_id, @telemetry_event_id, @alert_type, @started_at, @ended_at,
		    @operational_date::text::date, @shift_code, @threshold_value,
		    ST_SetSRID(ST_MakePoint(@longitude, @latitude), 4326), @speed_kph, @is_engine_on, @equipment_status,
		    @vessel_status, @mileage_m, @fuel_level_raw, @fuel_volume_liter, @fuel_percent, @fuel_delta_liter,
		    @fuel_temperature_c, @is_inside_geofence, @geofence_fid, @location_category, @segment_name, @operator_name
		)
		ON CONFLICT (equipment_id, alert_type) WHERE ended_at IS NULL DO NOTHING
		RETURNING id`, args).Scan(&a.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrAlertAlreadyOpen // satu alert terbuka per unit + jenis (uq_alerts_one_open)
	}
	return err
}

// InsertFuelEvent menyimpan satu kejadian fuel dan mengisi f.ID.
func (r *Repository) InsertFuelEvent(ctx context.Context, f *domain.FuelEvent) error {
	args := snapshotArgs(pgx.NamedArgs{
		"project_id": f.ProjectID, "equipment_id": f.EquipmentID, "device_id": f.DeviceID,
		"telemetry_event_id": f.TelemetryEventID, "event_type": f.EventType, "event_time": f.EventTime.UTC(),
		"operational_date": f.OperationalDate, "shift_code": f.ShiftCode, "fuel_delta_liter": f.FuelDeltaLiter,
		"description": f.Description, "alert_id": f.AlertID,
	}, f.Snapshot)
	return r.pool.QueryRow(ctx, `
		INSERT INTO iot.fuel_events (
		    project_id, equipment_id, device_id, telemetry_event_id, event_type, event_time, operational_date,
		    shift_code, fuel_delta_liter, fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_temperature_c,
		    location, speed_kph, is_engine_on, equipment_status, vessel_status, mileage_m,
		    is_inside_geofence, geofence_fid, location_category, segment_name, description, alert_id
		) VALUES (
		    @project_id, @equipment_id, @device_id, @telemetry_event_id, @event_type, @event_time,
		    @operational_date::text::date, @shift_code, @fuel_delta_liter, @fuel_level_raw, @fuel_volume_liter,
		    @fuel_percent, @fuel_temperature_c, ST_SetSRID(ST_MakePoint(@longitude, @latitude), 4326), @speed_kph,
		    @is_engine_on, @equipment_status, @vessel_status, @mileage_m, @is_inside_geofence, @geofence_fid,
		    @location_category, @segment_name, @description, @alert_id
		) RETURNING id`, args).Scan(&f.ID)
}

// CloseAlert menutup alert TERBUKA unit + jenis (ended_at = endedAt) dan mengembalikannya (nil bila tidak ada).
func (r *Repository) CloseAlert(ctx context.Context, equipmentID uuid.UUID, alertType string, endedAt time.Time) (*domain.Alert, error) {
	var a domain.Alert
	err := r.pool.QueryRow(ctx, `
		UPDATE iot.alerts SET ended_at = greatest($3, started_at), updated_at = now()
		 WHERE equipment_id = $1 AND alert_type = $2 AND ended_at IS NULL
		RETURNING id, project_id, equipment_id, device_id, coalesce(telemetry_event_id, 0), alert_type, started_at, ended_at,
		          operational_date::text, shift_code, threshold_value::float8, operator_name`,
		equipmentID, alertType, endedAt.UTC()).Scan(&a.ID, &a.ProjectID, &a.EquipmentID, &a.DeviceID, &a.TelemetryEventID,
		&a.AlertType, &a.StartedAt, &a.EndedAt, &a.OperationalDate, &a.ShiftCode, &a.ThresholdValue, &a.OperatorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ExtendAlert memperpanjang ended_at alert unit + jenis yang awalnya startedAt (aturan lama: underspeed & FUD
// diperpanjang selama kondisi masih berlangsung). nil bila tidak ada.
func (r *Repository) ExtendAlert(ctx context.Context, equipmentID uuid.UUID, alertType string, startedAt, endedAt time.Time) (*domain.Alert, error) {
	var a domain.Alert
	err := r.pool.QueryRow(ctx, `
		UPDATE iot.alerts SET ended_at = greatest($4, started_at), updated_at = now()
		 WHERE equipment_id = $1 AND alert_type = $2 AND started_at = $3
		RETURNING id, project_id, equipment_id, device_id, coalesce(telemetry_event_id, 0), alert_type, started_at, ended_at,
		          operational_date::text, shift_code, threshold_value::float8, operator_name`,
		equipmentID, alertType, startedAt.UTC(), endedAt.UTC()).Scan(&a.ID, &a.ProjectID, &a.EquipmentID, &a.DeviceID,
		&a.TelemetryEventID, &a.AlertType, &a.StartedAt, &a.EndedAt, &a.OperationalDate, &a.ShiftCode, &a.ThresholdValue, &a.OperatorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
