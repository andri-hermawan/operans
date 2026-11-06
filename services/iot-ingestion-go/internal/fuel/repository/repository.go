// Package repository = SQL BACA fitur fuel.
//
// Pengecualian yang disengaja (ADR 0011): iot.fuel_events DITULIS oleh repository fitur ingest (pipeline),
// dan hanya DIBACA di sini.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/fuel/domain"
	"simha/iot-ingestion/internal/shared/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// FuelEventsRange = kejadian fuel dalam rentang tanggal operasional (laporan Fuel History).
func (r *Repository) FuelEventsRange(ctx context.Context, f database.RangeFilter) ([]domain.FuelEventRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, equipment_id, event_type, event_time, fuel_delta_liter, fuel_volume_liter, description, alert_id
		  FROM iot.fuel_events
		 WHERE operational_date BETWEEN $1::date AND $2::date AND ($3::text IS NULL OR shift_code = $3)
		   AND ($4::uuid IS NULL OR project_id = $4) AND ($5::uuid IS NULL OR equipment_id = $5)
		 ORDER BY event_time, id LIMIT $6`, f.From, f.To, f.ShiftCode, f.ProjectID, f.EquipmentID, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FuelEventRow{}
	for rows.Next() {
		var x domain.FuelEventRow
		if err := rows.Scan(&x.ID, &x.EquipmentID, &x.EventType, &x.EventTime, &x.DeltaLiter, &x.Volume, &x.Description, &x.AlertID); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
