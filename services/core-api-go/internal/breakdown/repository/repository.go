// Package repository (fitur breakdown) = semua SQL ke tabel core.equipment_breakdowns
// (shift & label pilihan dibaca lewat JOIN ke core.shifts & core.reference_values).
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/breakdown/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const breakdownSQL = `
SELECT b.id, b.project_id, b.operational_date::text, b.shift_id, s.shift_code, s.shift_name,
       b.equipment_code, b.equipment_id, b.status,
       coalesce(b.category_code, ''), coalesce(c.label, b.category_code, ''),
       b.started_at, b.ended_at, b.duration_hours::float8,
       coalesce(b.repair_status_code, ''), coalesce(r.label, b.repair_status_code, ''),
       b.description, b.location, b.updated_at
  FROM core.equipment_breakdowns b
  JOIN core.shifts s ON s.id = b.shift_id
  LEFT JOIN core.reference_values c ON c.group_code = 'breakdown_category' AND c.code = b.category_code
  LEFT JOIN core.reference_values r ON r.group_code = 'repair_status'      AND r.code = b.repair_status_code`

func scanBreakdown(row pgx.Row) (*domain.EquipmentBreakdown, error) {
	var b domain.EquipmentBreakdown
	err := row.Scan(&b.ID, &b.ProjectID, &b.OperationalDate, &b.ShiftID, &b.ShiftCode, &b.ShiftName,
		&b.EquipmentCode, &b.EquipmentID, &b.Status, &b.CategoryCode, &b.CategoryLabel,
		&b.StartedAt, &b.EndedAt, &b.DurationHours, &b.RepairStatusCode, &b.RepairStatusLabel,
		&b.Description, &b.Location, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.EquipmentBreakdown, error) {
	rows, err := r.pool.Query(ctx, breakdownSQL+`
		 WHERE ($1::uuid IS NULL OR b.project_id = $1)
		   AND ($2::text = '' OR b.operational_date >= $2::text::date)
		   AND ($3::text = '' OR b.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR b.shift_id = $4)
		   AND ($5::text = '' OR b.equipment_code = $5)
		 ORDER BY b.operational_date DESC, s.sequence, b.equipment_code, b.started_at NULLS LAST
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.EquipmentBreakdown{}
	for rows.Next() {
		b, err := scanBreakdown(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.EquipmentBreakdown, error) {
	return scanBreakdown(r.pool.QueryRow(ctx, breakdownSQL+` WHERE b.id = $1`, id))
}

// Kunci "data sama ditimpa" (ADR 0007): tanggal + shift + unit + kategori + jam mulai.
const upsertBreakdownSQL = `
INSERT INTO core.equipment_breakdowns (project_id, operational_date, shift_id, equipment_code, equipment_id, status,
       category_code, started_at, ended_at, duration_hours, repair_status_code, description, location, created_by, updated_by)
VALUES ($1, $2::text::date, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
ON CONFLICT ON CONSTRAINT uq_equipment_breakdowns_key
DO UPDATE SET equipment_id = EXCLUDED.equipment_id, status = EXCLUDED.status, ended_at = EXCLUDED.ended_at,
              duration_hours = EXCLUDED.duration_hours, repair_status_code = EXCLUDED.repair_status_code,
              description = EXCLUDED.description, location = EXCLUDED.location, updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`

func breakdownArgs(b domain.EquipmentBreakdown, by uuid.UUID) []any {
	return []any{b.ProjectID, b.OperationalDate, b.ShiftID, b.EquipmentCode, b.EquipmentID, b.Status,
		b.CategoryCode, b.StartedAt, b.EndedAt, b.DurationHours, b.RepairStatusCode, b.Description, b.Location, by}
}

func (r *Repository) Upsert(ctx context.Context, b domain.EquipmentBreakdown, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertBreakdownSQL, breakdownArgs(b, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, b domain.EquipmentBreakdown, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.equipment_breakdowns
		   SET operational_date = $2::text::date, shift_id = $3, equipment_code = $4, equipment_id = $5, status = $6,
		       category_code = $7, started_at = $8, ended_at = $9, duration_hours = $10, repair_status_code = $11,
		       description = $12, location = $13, updated_by = $14
		 WHERE id = $1`,
		b.ID, b.OperationalDate, b.ShiftID, b.EquipmentCode, b.EquipmentID, b.Status, b.CategoryCode,
		b.StartedAt, b.EndedAt, b.DurationHours, b.RepairStatusCode, b.Description, b.Location, by)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: tanggal, shift, unit, kategori & jam mulai sama", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.equipment_breakdowns WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.EquipmentBreakdown, by uuid.UUID, dryRun bool) (int, int, error) {
	args := make([][]any, len(rows))
	for i, b := range rows {
		args[i] = breakdownArgs(b, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertBreakdownSQL, args, dryRun)
}
