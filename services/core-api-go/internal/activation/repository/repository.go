// Package repository (fitur activation) = semua SQL ke tabel core.equipment_activations
// (kode unit & project dibaca lewat JOIN ke core.equipment).
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/activation/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const activationSQL = `
SELECT a.id, a.equipment_id, e.equipment_code, e.project_id, a.activation_date::text, a.status, a.description, a.updated_at
  FROM core.equipment_activations a
  JOIN core.equipment e ON e.id = a.equipment_id`

func scanActivation(row pgx.Row) (*domain.Activation, error) {
	var a domain.Activation
	err := row.Scan(&a.ID, &a.EquipmentID, &a.EquipmentCode, &a.ProjectID, &a.ActivationDate, &a.Status,
		&a.Description, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// List: semua filter opsional. Maks. 2000 baris, terbaru dulu.
func (r *Repository) List(ctx context.Context, f domain.ActivationFilter) ([]domain.Activation, error) {
	rows, err := r.pool.Query(ctx, activationSQL+`
		 WHERE ($1::uuid IS NULL OR e.project_id = $1)
		   AND ($2::uuid IS NULL OR a.equipment_id = $2)
		   AND ($3::text = '' OR a.activation_date >= $3::text::date)
		   AND ($4::text = '' OR a.activation_date <= $4::text::date)
		   AND ($5::text = '' OR a.status = $5)
		 ORDER BY a.activation_date DESC, e.equipment_code
		 LIMIT 2000`, f.ProjectID, f.EquipmentID, f.DateFrom, f.DateTo, f.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Activation{}
	for rows.Next() {
		a, err := scanActivation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.Activation, error) {
	return scanActivation(r.pool.QueryRow(ctx, activationSQL+` WHERE a.id = $1`, id))
}

// Upsert: satu baris per equipment per tanggal; bila sudah ada → diperbarui.
// created=true bila baris baru (xmax = 0 adalah cara PostgreSQL membedakan INSERT dari UPDATE).
func (r *Repository) Upsert(ctx context.Context, equipmentID uuid.UUID, date, status string, desc *string, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.equipment_activations (equipment_id, activation_date, status, description, created_by, updated_by)
		VALUES ($1, $2::text::date, $3, $4, $5, $5)
		ON CONFLICT (equipment_id, activation_date)
		DO UPDATE SET status = EXCLUDED.status, description = EXCLUDED.description, updated_by = EXCLUDED.updated_by
		RETURNING id, (xmax = 0)`, equipmentID, date, status, desc, by).Scan(&id, &created)
	return id, created, err
}

// Update mengubah kolom yang tidak nil.
func (r *Repository) Update(ctx context.Context, id int64, date, status, desc *string, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.equipment_activations
		   SET activation_date = COALESCE($2::text::date, activation_date),
		       status          = COALESCE($3, status),
		       description     = COALESCE($4, description),
		       updated_by      = $5
		 WHERE id = $1`, id, date, status, desc, by)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: equipment & tanggal sama", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.equipment_activations WHERE id = $1`, id)
	return err
}

// BulkUpsert: aktivasi harian dari Excel (kunci equipment + tanggal).
func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.ActivationRow, by uuid.UUID, dryRun bool) (created, updated int, err error) {
	args := make([][]any, len(rows))
	for i, a := range rows {
		args[i] = []any{a.EquipmentID, a.Date, a.Status, a.Description, by}
	}
	return database.BulkUpsert(ctx, r.pool, `
		INSERT INTO core.equipment_activations (equipment_id, activation_date, status, description, created_by, updated_by)
		VALUES ($1, $2::text::date, $3, $4, $5, $5)
		ON CONFLICT (equipment_id, activation_date)
		DO UPDATE SET status = EXCLUDED.status, description = EXCLUDED.description, updated_by = EXCLUDED.updated_by
		RETURNING id, (xmax = 0)`, args, dryRun)
}
