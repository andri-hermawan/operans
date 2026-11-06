// Package repository = SQL BACA fitur telemetry.
//
// Pengecualian yang disengaja (ADR 0011): iot.telemetry_events DITULIS oleh repository fitur ingest
// (pipeline), dan hanya DIBACA di sini untuk halaman dashboard & pemantauan.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/shared/database"
	"simha/iot-ingestion/internal/telemetry/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Semua fungsi baca menerima projectID:
//   - nil       → semua project (khusus superadmin)
//   - bukan nil → hanya baris milik project itu

// Daftar kolom TETAP (bukan input pengguna), jadi aman disambung ke query.
// Urutannya HARUS sama dengan urutan Scan di scanTelemetry.
const telemetryCols = `
    id, event_id, device_id, equipment_id, project_id, event_time, created_at,
    ST_Y(location), ST_X(location), altitude_m, heading_deg, speed_kph, satellite_count, hdop, pdop, gnss_status,
    is_engine_on, is_moving, mileage_m, odometer_m,
    fuel_level_raw, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
    equipment_status, vessel_status, vessel_load_ton, location_category, segment_name,
    operational_date::text, shift_code, operator_name, is_breakdown, is_inside_geofence, geofence_fid`

func scanTelemetry(row pgx.Row, x *domain.TelemetryRow) error {
	return row.Scan(
		&x.ID, &x.EventID, &x.DeviceID, &x.EquipmentID, &x.ProjectID, &x.EventTime, &x.CreatedAt,
		&x.Latitude, &x.Longitude, &x.AltitudeM, &x.HeadingDeg, &x.SpeedKph, &x.SatelliteCount, &x.HDOP, &x.PDOP, &x.GNSSStatus,
		&x.IsEngineOn, &x.IsMoving, &x.MileageM, &x.OdometerM,
		&x.FuelLevelRaw, &x.FuelVolumeLiter, &x.FuelPercent, &x.FuelDeltaLiter, &x.FuelTemperatureC,
		&x.EquipmentStatus, &x.VesselStatus, &x.VesselLoadTon, &x.LocationCategory, &x.SegmentName,
		&x.OperationalDate, &x.ShiftCode, &x.OperatorName, &x.IsBreakdown, &x.IsInsideGeofence, &x.GeofenceFID,
	)
}

// LatestByEquipment = data paling baru satu equipment. (nil, nil) bila belum ada data.
func (r *Repository) LatestByEquipment(ctx context.Context, projectID *uuid.UUID, equipmentID uuid.UUID) (*domain.TelemetryRow, error) {
	var x domain.TelemetryRow
	err := scanTelemetry(r.pool.QueryRow(ctx,
		`SELECT `+telemetryCols+`
		   FROM iot.telemetry_events
		  WHERE equipment_id = $1
		    AND ($2::uuid IS NULL OR project_id = $2)
		  ORDER BY event_time DESC
		  LIMIT 1`,
		equipmentID, projectID), &x)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// HistoryByEquipment = riwayat satu equipment dalam rentang [from, to), terbaru dulu, maksimal limit baris.
// Batas rentang & limit WAJIB dicek pemanggil (handler) agar query tidak membebani database.
func (r *Repository) HistoryByEquipment(ctx context.Context, projectID *uuid.UUID, equipmentID uuid.UUID, from, to time.Time, limit int) ([]domain.TelemetryRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+telemetryCols+`
		   FROM iot.telemetry_events
		  WHERE equipment_id = $1
		    AND ($2::uuid IS NULL OR project_id = $2)
		    AND event_time >= $3 AND event_time < $4
		  ORDER BY event_time DESC
		  LIMIT $5`,
		equipmentID, projectID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.TelemetryRow{} // slice kosong (bukan nil) → JSON [] bukan null
	for rows.Next() {
		var x domain.TelemetryRow
		if err := scanTelemetry(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Query baca untuk halaman pemantauan (Fase D/F3): filter database.Slot (tanggal operasional + shift).

// Points: titik urut waktu. stepSec > 0 = ambil satu titik per unit per stepSec detik (heatmap seluruh unit).
func (r *Repository) Points(ctx context.Context, s database.Slot, stepSec, limit int) ([]domain.Point, error) {
	q := `SELECT id, equipment_id, event_time, ST_Y(location), ST_X(location), speed_kph, heading_deg, is_engine_on,
	             equipment_status, vessel_status, fuel_volume_liter, fuel_percent, fuel_delta_liter, fuel_temperature_c,
	             segment_name, location_category, is_inside_geofence, operator_name, is_breakdown, shift_code,
	             operational_date::text
	        FROM iot.telemetry_events WHERE equipment_id IS NOT NULL AND ` + database.SlotWhere
	if stepSec > 0 {
		q = `SELECT * FROM (SELECT DISTINCT ON (equipment_id, floor(extract(epoch FROM event_time) / $6)) * FROM (` + q +
			`) t ORDER BY equipment_id, floor(extract(epoch FROM event_time) / $6), event_time) d ORDER BY event_time LIMIT $5`
	} else {
		q += ` ORDER BY event_time, id LIMIT $5`
	}
	args := append(s.Args(), limit)
	if stepSec > 0 {
		args = append(args, stepSec)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Point{}
	for rows.Next() {
		var p domain.Point
		if err := rows.Scan(&p.ID, &p.EquipmentID, &p.EventTime, &p.Latitude, &p.Longitude, &p.SpeedKph, &p.HeadingDeg,
			&p.IsEngineOn, &p.EquipmentStatus, &p.VesselStatus, &p.FuelVolume, &p.FuelPercent, &p.FuelDelta,
			&p.FuelTemperature, &p.SegmentName, &p.Category, &p.IsInside, &p.OperatorName, &p.IsBreakdown,
			&p.ShiftCode, &p.OperationalDate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) SegmentSpeeds(ctx context.Context, s database.Slot) ([]domain.SegmentSpeed, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT segment_name,
		       round(avg(speed_kph) FILTER (WHERE vessel_status = 'empty'), 1),
		       round(avg(speed_kph) FILTER (WHERE vessel_status = 'loaded'), 1),
		       round(avg(speed_kph), 1), count(*)
		  FROM iot.telemetry_events
		 WHERE `+database.SlotWhere+` AND speed_kph > 0 AND is_inside_geofence AND segment_name IS NOT NULL
		 GROUP BY segment_name ORDER BY segment_name`, s.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SegmentSpeed{}
	for rows.Next() {
		var x domain.SegmentSpeed
		if err := rows.Scan(&x.SegmentName, &x.AvgEmpty, &x.AvgLoaded, &x.AvgAll, &x.Points); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *Repository) Activities(ctx context.Context, s database.Slot) ([]domain.Activity, error) {
	rows, err := r.pool.Query(ctx, `
		WITH t AS (
		  SELECT equipment_id, event_time, equipment_status st, vessel_status vs, speed_kph sp, mileage_m mi,
		         fuel_volume_liter fv, fuel_delta_liter fd,
		         extract(epoch FROM lead(event_time) OVER (PARTITION BY equipment_id ORDER BY event_time) - event_time) / 3600 dur
		    FROM iot.telemetry_events WHERE equipment_id IS NOT NULL AND `+database.SlotWhere+`)
		SELECT equipment_id,
		  coalesce(round(sum(dur) FILTER (WHERE st = 'running' AND vs IN ('empty','loaded'))::numeric, 2), 0),
		  coalesce(round(sum(dur) FILTER (WHERE st = 'running' AND vs = 'empty')::numeric, 2), 0),
		  coalesce(round(sum(dur) FILTER (WHERE st = 'running' AND vs = 'loaded')::numeric, 2), 0),
		  coalesce(round(sum(dur) FILTER (WHERE st = 'idle' AND vs IN ('empty','loaded'))::numeric, 2), 0),
		  coalesce(round(sum(dur) FILTER (WHERE st = 'idle' AND vs = 'empty')::numeric, 2), 0),
		  coalesce(round(sum(dur) FILTER (WHERE st = 'idle' AND vs = 'loaded')::numeric, 2), 0),
		  coalesce(round(avg(sp) FILTER (WHERE st = 'running' AND sp > 0), 2), 0),
		  coalesce(round(avg(sp) FILTER (WHERE st = 'running' AND sp > 0 AND vs = 'empty'), 2), 0),
		  coalesce(round(avg(sp) FILTER (WHERE st = 'running' AND sp > 0 AND vs = 'loaded'), 2), 0),
		  coalesce(max(sp) FILTER (WHERE st = 'running' AND sp > 0), 0),
		  coalesce(max(sp) FILTER (WHERE st = 'running' AND sp > 0 AND vs = 'empty'), 0),
		  coalesce(max(sp) FILTER (WHERE st = 'running' AND sp > 0 AND vs = 'loaded'), 0),
		  coalesce(round(((max(mi) - min(mi)) / 1000.0)::numeric, 2), 0),
		  (array_agg(fv ORDER BY event_time) FILTER (WHERE fv IS NOT NULL))[1],
		  (array_agg(fv ORDER BY event_time DESC) FILTER (WHERE fv IS NOT NULL))[1],
		  coalesce(round(sum(fd) FILTER (WHERE fd > 0), 2), 0),
		  coalesce(round(sum(-fd) FILTER (WHERE fd < 0), 2), 0),
		  count(*)
		FROM t GROUP BY equipment_id ORDER BY equipment_id`, s.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Activity{}
	for rows.Next() {
		var a domain.Activity
		if err := rows.Scan(&a.EquipmentID, &a.RunningHours, &a.RunningEmpty, &a.RunningLoaded, &a.IdleHours, &a.IdleEmpty,
			&a.IdleLoaded, &a.AvgSpeed, &a.AvgSpeedEmpty, &a.AvgSpeedLoaded, &a.MaxSpeed, &a.MaxSpeedEmpty, &a.MaxSpeedLoaded,
			&a.MileageKm, &a.FuelStart, &a.FuelRemaining, &a.FuelIncrease, &a.FuelDecrease, &a.Points); err != nil {
			return nil, err
		}
		if a.FuelDecrease > 0 {
			a.FuelBurnKmPerLtr = float64(int(a.MileageKm/a.FuelDecrease*100+0.5)) / 100
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
