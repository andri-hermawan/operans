// Package repository (fitur operator) = semua SQL ke tabel core.operator_assignments
// (kode & nama shift dibaca lewat JOIN ke core.shifts).
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/operator/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const operatorSQL = `
SELECT o.id, o.project_id, o.operational_date::text, o.shift_id, s.shift_code, s.shift_name,
       o.equipment_code, o.equipment_id, o.operator_name, o.description, o.updated_at
  FROM core.operator_assignments o
  JOIN core.shifts s ON s.id = o.shift_id`

func scanOperator(row pgx.Row) (*domain.OperatorAssignment, error) {
	var o domain.OperatorAssignment
	err := row.Scan(&o.ID, &o.ProjectID, &o.OperationalDate, &o.ShiftID, &o.ShiftCode, &o.ShiftName,
		&o.EquipmentCode, &o.EquipmentID, &o.OperatorName, &o.Description, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.OperatorAssignment, error) {
	rows, err := r.pool.Query(ctx, operatorSQL+`
		 WHERE ($1::uuid IS NULL OR o.project_id = $1)
		   AND ($2::text = '' OR o.operational_date >= $2::text::date)
		   AND ($3::text = '' OR o.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR o.shift_id = $4)
		   AND ($5::text = '' OR o.equipment_code = $5)
		 ORDER BY o.operational_date DESC, s.sequence, o.equipment_code
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.OperatorAssignment{}
	for rows.Next() {
		o, err := scanOperator(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.OperatorAssignment, error) {
	return scanOperator(r.pool.QueryRow(ctx, operatorSQL+` WHERE o.id = $1`, id))
}

const upsertOperatorSQL = `
INSERT INTO core.operator_assignments (project_id, operational_date, shift_id, equipment_code, equipment_id,
                                       operator_name, description, created_by, updated_by)
VALUES ($1, $2::text::date, $3, $4, $5, $6, $7, $8, $8)
ON CONFLICT (project_id, operational_date, shift_id, equipment_code)
DO UPDATE SET operator_name = EXCLUDED.operator_name, description = EXCLUDED.description,
              equipment_id = EXCLUDED.equipment_id, updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`

func operatorArgs(o domain.OperatorAssignment, by uuid.UUID) []any {
	return []any{o.ProjectID, o.OperationalDate, o.ShiftID, o.EquipmentCode, o.EquipmentID, o.OperatorName, o.Description, by}
}

// Upsert: tanggal + shift + unit sama → diperbarui. created=true bila baris baru.
func (r *Repository) Upsert(ctx context.Context, o domain.OperatorAssignment, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertOperatorSQL, operatorArgs(o, by)...).Scan(&id, &created)
	return id, created, err
}

// Update menyimpan seluruh isi o (sudah digabung dengan nilai lama oleh service).
func (r *Repository) Update(ctx context.Context, o domain.OperatorAssignment, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.operator_assignments
		   SET operational_date = $2::text::date, shift_id = $3, equipment_code = $4, equipment_id = $5,
		       operator_name = $6, description = $7, updated_by = $8
		 WHERE id = $1`,
		o.ID, o.OperationalDate, o.ShiftID, o.EquipmentCode, o.EquipmentID, o.OperatorName, o.Description, by)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: tanggal, shift & unit sama", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.operator_assignments WHERE id = $1`, id)
	return err
}

// BulkUpsert menyimpan semua baris dalam SATU transaksi (dikirim sekaligus = cepat).
// dryRun=true → dijalankan lalu DIBATALKAN, hanya untuk menghitung baru vs diperbarui.
func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.OperatorAssignment, by uuid.UUID, dryRun bool) (created, updated int, err error) {
	args := make([][]any, len(rows))
	for i, o := range rows {
		args[i] = operatorArgs(o, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertOperatorSQL, args, dryRun)
}
