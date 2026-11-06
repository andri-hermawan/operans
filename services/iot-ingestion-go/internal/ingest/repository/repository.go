// Package repository = SQL yang DITULIS pipeline ingest: iot.telemetry_events, iot.alerts, iot.fuel_events,
// iot.geofence_events.
//
// Pengecualian yang disengaja (ADR 0011): tabel-tabel ini DITULIS di sini (pipeline), tetapi DIBACA oleh
// repository fitur lain untuk halaman pemantauan — telemetry (iot.telemetry_events), alert (iot.alerts),
// fuel (iot.fuel_events), geofence (iot.geofence_events). Menulis hanya di sini; membaca hanya di sana.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ---------------------------------------------------------------------
// MENULIS: simpan satu event (idempotent)
// ---------------------------------------------------------------------

// Kolom kelompok 1–6 + metadata dari alat, plus kolom yang dihitung server (domain.Derived).
// Kolom fuel liter menyusul di Fase C3.
const insertEventSQL = `
INSERT INTO iot.telemetry_events (
    event_id, device_id, equipment_id, project_id, event_time,
    location, altitude_m, heading_deg, speed_kph, satellite_count, hdop, pdop, gnss_status,
    is_engine_on, is_moving, mileage_m, odometer_m, vessel_load_ton,
    fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
    external_voltage_v, battery_voltage_v, battery_current_a, gsm_signal_level, gsm_operator_code, sleep_mode,
    accel_x_mg, accel_y_mg, accel_z_mg, analog_input_1_v,
    equipment_status, operational_date, shift_code, operator_name, is_breakdown, breakdown_description,
    zone_id, is_inside_geofence, geofence_fid, location_category, segment_name, vessel_status,
    metadata
) VALUES (
    @event_id, @device_id, @equipment_id, @project_id, @event_time,
    ST_SetSRID(ST_MakePoint(@longitude, @latitude), 4326), @altitude_m, @heading_deg, @speed_kph, @satellite_count, @hdop, @pdop, @gnss_status,
    @is_engine_on, @is_moving, @mileage_m, @odometer_m, @vessel_load_ton,
    @fuel_level_raw, @fuel_volume_liter, @fuel_percent, @fuel_delta_liter, @fuel_temperature_c,
    @external_voltage_v, @battery_voltage_v, @battery_current_a, @gsm_signal_level, @gsm_operator_code, @sleep_mode,
    @accel_x_mg, @accel_y_mg, @accel_z_mg, @analog_input_1_v,
    @equipment_status, @operational_date::text::date, @shift_code, @operator_name, @is_breakdown, @breakdown_description,
    @zone_id, @is_inside_geofence, @geofence_fid, @location_category, @segment_name, @vessel_status,
    @metadata
)
ON CONFLICT (device_id, event_id) DO NOTHING
RETURNING id, created_at`

// Saved = hasil menyimpan satu event.
type Saved struct {
	ID         int64     // iot.telemetry_events.id
	ReceivedAt time.Time // created_at = waktu diterima server
}

// InsertEvent menyimpan event milik alat dev.
// inserted=false (tanpa error) berarti event ini DUPLIKAT dan diabaikan (saved kosong).
func (r *Repository) InsertEvent(ctx context.Context, dev *devicedomain.Device, e *domain.Event) (saved Saved, inserted bool, err error) {
	meta := e.Metadata
	if meta == nil {
		meta = map[string]any{} // map nil akan menjadi JSON null; kolom kita minta '{}'
	}

	err = r.pool.QueryRow(ctx, insertEventSQL, pgx.NamedArgs{
		// 1. Identitas & waktu — device, equipment & project dari tabel devices, BUKAN dari kiriman alat
		"event_id":     e.EventID,
		"device_id":    dev.ID,
		"equipment_id": dev.EquipmentID,
		"project_id":   dev.ProjectID,
		"event_time":   e.EventTime.UTC(),
		// 2. Posisi GNSS
		"latitude":        e.Latitude,
		"longitude":       e.Longitude,
		"altitude_m":      e.AltitudeM,
		"heading_deg":     e.HeadingDeg,
		"speed_kph":       e.SpeedKph,
		"satellite_count": e.SatelliteCount,
		"hdop":            e.HDOP,
		"pdop":            e.PDOP,
		"gnss_status":     e.GNSSStatus,
		// 3. Mesin & jarak
		"is_engine_on":    e.IsEngineOn,
		"is_moving":       e.IsMoving,
		"mileage_m":       e.MileageM,
		"odometer_m":      e.OdometerM,
		"vessel_load_ton": e.VesselLoadTon,
		// 4. Fuel
		"fuel_level_raw":     e.FuelLevelRaw,
		"fuel_volume_liter":  e.FuelVolumeLiter,
		"fuel_percent":       e.FuelPercent,
		"fuel_delta_liter":   e.FuelDeltaLiter,
		"fuel_temperature_c": e.FuelTemperatureC,
		// 5. Kelistrikan & modem
		"external_voltage_v": e.ExternalVoltageV,
		"battery_voltage_v":  e.BatteryVoltageV,
		"battery_current_a":  e.BatteryCurrentA,
		"gsm_signal_level":   e.GSMSignalLevel,
		"gsm_operator_code":  e.GSMOperatorCode,
		"sleep_mode":         e.SleepMode,
		// 6. Sensor tambahan
		"accel_x_mg":       e.AccelXmg,
		"accel_y_mg":       e.AccelYmg,
		"accel_z_mg":       e.AccelZmg,
		"analog_input_1_v": e.AnalogInput1V,
		// Dihitung server
		"equipment_status":      e.EquipmentStatus,
		"operational_date":      e.OperationalDate,
		"shift_code":            e.ShiftCode,
		"operator_name":         e.OperatorName,
		"is_breakdown":          e.IsBreakdown,
		"breakdown_description": e.BreakdownDescription,
		"zone_id":               e.ZoneID,
		"is_inside_geofence":    e.IsInsideGeofence,
		"geofence_fid":          e.GeofenceFID,
		"location_category":     e.LocationCategory,
		"segment_name":          e.SegmentName,
		"vessel_status":         e.VesselStatus,
		// 9. Data vendor lain
		"metadata": meta,
	}).Scan(&saved.ID, &saved.ReceivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Saved{}, false, nil // ON CONFLICT DO NOTHING → tidak ada baris yang dikembalikan = duplikat
	}
	if err != nil {
		return Saved{}, false, err
	}
	return saved, true, nil
}
