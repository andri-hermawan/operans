// Package repository = SQL BACA fitur geofence.
//
// Pengecualian yang disengaja (ADR 0011): iot.geofence_events DITULIS oleh repository fitur ingest (pipeline),
// dan hanya DIBACA di sini.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/geofence/domain"
	"simha/iot-ingestion/internal/shared/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) GeofenceEvents(ctx context.Context, s database.Slot, segment *string) ([]domain.GeofenceRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, equipment_id, event_type, event_time, shift_code, segment_name, location_category,
		       ST_Y(location), ST_X(location), speed_kph, vessel_status
		  FROM iot.geofence_events
		 WHERE `+database.SlotWhere+` AND ($5::text IS NULL OR segment_name = $5)
		 ORDER BY event_time, id LIMIT 20000`, append(s.Args(), segment)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GeofenceRow{}
	for rows.Next() {
		var g domain.GeofenceRow
		if err := rows.Scan(&g.ID, &g.EquipmentID, &g.EventType, &g.EventTime, &g.ShiftCode, &g.SegmentName, &g.Category,
			&g.Latitude, &g.Longitude, &g.SpeedKph, &g.Vessel); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
