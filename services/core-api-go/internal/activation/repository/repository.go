// Package repository (fitur activation) = semua SQL ke tabel core.equipment_activations
// (kode unit & project disimpan di tabel itu sendiri, seperti weighbridge_tickets).
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
SELECT a.id, a.project_id, a.equipment_id, a.equipment_code, a.activation_date::text, a.status, a.description, a.updated_at
  FROM core.equipment_activations a`

// Kunci "data sama ditimpa": project + kode unit + tanggal.
const upsertSQL = `
INSERT INTO core.equipment_activations (project_id, equipment_code, equipment_id, activation_date, status, description, created_by, updated_by)
VALUES ($1, $2, $3, $4::text::date, $5, $6, $7, $7)
ON CONFLICT ON CONSTRAINT uq_equipment_activations_key
DO UPDATE SET equipment_id = EXCLUDED.equipment_id, status = EXCLUDED.status, description = EXCLUDED.description, updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`

func scanActivation(row pgx.Row) (*domain.Activation, error) {
	var a domain.Activation
	err := row.Scan(&a.ID, &a.ProjectID, &a.EquipmentID, &a.EquipmentCode, &a.ActivationDate, &a.Status, &a.Description, &a.UpdatedAt)
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
		 WHERE ($1::uuid IS NULL OR a.project_id = $1)
		   AND ($2::text = '' OR a.equipment_code = $2)
		   AND ($3::text = '' OR a.activation_date >= $3::text::date)
		   AND ($4::text = '' OR a.activation_date <= $4::text::date)
		   AND ($5::text = '' OR a.status = $5)
		 ORDER BY a.activation_date DESC, a.equipment_code
		 LIMIT 2000`, f.ProjectID, f.EquipmentCode, f.DateFrom, f.DateTo, f.Status)
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

func args(a domain.ActivationRow, by uuid.UUID) []any {
	return []any{a.ProjectID, a.EquipmentCode, a.EquipmentID, a.Date, a.Status, a.Description, by}
}

// Upsert: satu baris per unit per tanggal; bila sudah ada → diperbarui.
// created=true bila baris baru (xmax = 0 adalah cara PostgreSQL membedakan INSERT dari UPDATE).
func (r *Repository) Upsert(ctx context.Context, a domain.ActivationRow, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(a, by)...).Scan(&id, &created)
	return id, created, err
}

// Update mengubah kolom yang tidak nil (unit & project tidak bisa diganti).
func (r *Repository) Update(ctx context.Context, id int64, date, status, desc *string, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.equipment_activations
		   SET activation_date = COALESCE($2::text::date, activation_date),
		       status          = COALESCE($3, status),
		       description     = COALESCE($4, description),
		       updated_by      = $5
		 WHERE id = $1`, id, date, status, desc, by)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: unit & tanggal sama", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.equipment_activations WHERE id = $1`, id)
	return err
}

// BulkUpsert: aktivasi harian dari Excel (kunci project + kode unit + tanggal).
func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.ActivationRow, by uuid.UUID, dryRun bool) (created, updated int, err error) {
	a := make([][]any, len(rows))
	for i, row := range rows {
		a[i] = args(row, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
