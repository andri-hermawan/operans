// Package repository (fitur alerttype) = SQL ke tabel core.alert_types (jenis alert + nilai bawaan).
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/alerttype/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) List(ctx context.Context) ([]domain.AlertType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, legacy_code, name, description, uses_duration, uses_speed_limit, uses_fuel_drop,
		       default_duration_minutes::float8, default_speed_limit_kph::float8, default_fuel_drop_liters::float8
		  FROM core.alert_types ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AlertType{}
	for rows.Next() {
		var t domain.AlertType
		if err := rows.Scan(&t.Code, &t.LegacyCode, &t.Name, &t.Description, &t.UsesDuration, &t.UsesSpeedLimit,
			&t.UsesFuelDrop, &t.DefaultDurationMinutes, &t.DefaultSpeedLimitKph, &t.DefaultFuelDropLiters); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Update menyimpan nilai bawaan & teks jenis alert (sudah divalidasi service).
func (r *Repository) Update(ctx context.Context, t domain.AlertType, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.alert_types SET name = $2, description = $3, default_duration_minutes = $4,
		       default_speed_limit_kph = $5, default_fuel_drop_liters = $6, updated_by = $7
		 WHERE code = $1`,
		t.Code, t.Name, t.Description, t.DefaultDurationMinutes, t.DefaultSpeedLimitKph, t.DefaultFuelDropLiters, by)
	return err
}
