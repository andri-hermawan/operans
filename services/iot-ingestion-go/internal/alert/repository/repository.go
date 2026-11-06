// Package repository = SQL BACA fitur alert.
//
// Pengecualian yang disengaja (ADR 0011): iot.alerts DITULIS oleh repository fitur ingest (pipeline),
// dan hanya DIBACA di sini.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/alert/domain"
	"simha/iot-ingestion/internal/shared/database"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Alerts(ctx context.Context, f database.RangeFilter) ([]domain.AlertRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, equipment_id, alert_type, started_at, ended_at, operational_date::text, shift_code, threshold_value,
		       ST_Y(location), ST_X(location), speed_kph, is_engine_on, vessel_status, fuel_volume_liter, fuel_percent,
		       fuel_delta_liter, fuel_temperature_c, segment_name, location_category, operator_name
		  FROM iot.alerts
		 WHERE operational_date BETWEEN $1::date AND $2::date AND ($3::text IS NULL OR shift_code = $3)
		   AND ($4::uuid IS NULL OR project_id = $4) AND ($5::uuid IS NULL OR equipment_id = $5)
		   AND (cardinality($6::text[]) = 0 OR alert_type = ANY($6))
		 ORDER BY started_at, id LIMIT $7`,
		f.From, f.To, f.ShiftCode, f.ProjectID, f.EquipmentID, f.Types, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AlertRow{}
	for rows.Next() {
		var a domain.AlertRow
		if err := rows.Scan(&a.ID, &a.EquipmentID, &a.AlertType, &a.StartedAt, &a.EndedAt, &a.OperationalDate, &a.ShiftCode,
			&a.Threshold, &a.Latitude, &a.Longitude, &a.SpeedKph, &a.IsEngineOn, &a.VesselStatus, &a.FuelVolume,
			&a.FuelPercent, &a.FuelDelta, &a.FuelTemperature, &a.SegmentName, &a.Category, &a.OperatorName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
