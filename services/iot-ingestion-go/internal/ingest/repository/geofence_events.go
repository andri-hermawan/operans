package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"simha/iot-ingestion/internal/ingest/domain"
)

// InsertGeofenceEvent menyimpan satu kejadian masuk/keluar zona dan mengisi g.ID.
func (r *Repository) InsertGeofenceEvent(ctx context.Context, g *domain.GeofenceEvent) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO iot.geofence_events (
		    project_id, equipment_id, device_id, telemetry_event_id, event_type, event_time,
		    operational_date, shift_code, zone_id, geofence_fid, location_category, segment_name,
		    location, speed_kph, is_engine_on, equipment_status, vessel_status, fuel_level_raw, mileage_m
		) VALUES (
		    @project_id, @equipment_id, @device_id, @telemetry_event_id, @event_type, @event_time,
		    @operational_date::text::date, @shift_code, @zone_id, @geofence_fid, @location_category, @segment_name,
		    ST_SetSRID(ST_MakePoint(@longitude, @latitude), 4326), @speed_kph, @is_engine_on, @equipment_status,
		    @vessel_status, @fuel_level_raw, @mileage_m
		) RETURNING id`,
		pgx.NamedArgs{
			"project_id": g.ProjectID, "equipment_id": g.EquipmentID, "device_id": g.DeviceID,
			"telemetry_event_id": g.TelemetryEventID, "event_type": g.EventType, "event_time": g.EventTime.UTC(),
			"operational_date": g.OperationalDate, "shift_code": g.ShiftCode, "zone_id": g.ZoneID,
			"geofence_fid": g.GeofenceFID, "location_category": g.LocationCategory, "segment_name": g.SegmentName,
			"latitude": g.Latitude, "longitude": g.Longitude, "speed_kph": g.SpeedKph, "is_engine_on": g.IsEngineOn,
			"equipment_status": g.EquipmentStatus, "vessel_status": g.VesselStatus, "fuel_level_raw": g.FuelLevelRaw,
			"mileage_m": g.MileageM,
		}).Scan(&g.ID)
}
