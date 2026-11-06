// Package repository (fitur shift) = semua SQL ke tabel core.shifts.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/shift/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ListActive = shift aktif project, urut sequence. Jam dalam format HH:MM.
func (r *Repository) ListActive(ctx context.Context, projectID uuid.UUID) ([]domain.Shift, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, shift_code, shift_name, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'), sequence
		  FROM core.shifts WHERE project_id = $1 AND is_active ORDER BY sequence`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Shift{}
	for rows.Next() {
		var s domain.Shift
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Code, &s.Name, &s.Start, &s.End, &s.Sequence); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------
// Kelola shift (termasuk yang nonaktif)
// ---------------------------------------------------------------------

const shiftCols = `id, project_id, shift_code, shift_name, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
    sequence, is_active, updated_at`

func scanShiftRow(row pgx.Row) (*domain.ShiftRow, error) {
	var s domain.ShiftRow
	err := row.Scan(&s.ID, &s.ProjectID, &s.Code, &s.Name, &s.Start, &s.End, &s.Sequence, &s.IsActive, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) ListAll(ctx context.Context, projectID uuid.UUID) ([]domain.ShiftRow, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+shiftCols+` FROM core.shifts WHERE project_id = $1 ORDER BY sequence`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ShiftRow{}
	for rows.Next() {
		s, err := scanShiftRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.ShiftRow, error) {
	return scanShiftRow(r.pool.QueryRow(ctx, `SELECT `+shiftCols+` FROM core.shifts WHERE id = $1`, id))
}

func (r *Repository) Insert(ctx context.Context, s domain.ShiftRow, by uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.shifts (project_id, shift_code, shift_name, start_time, end_time, sequence, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4::time, $5::time, $6, $7, $8, $8) RETURNING id`,
		s.ProjectID, s.Code, s.Name, s.Start, s.End, s.Sequence, s.IsActive, by).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, apperror.ErrCodeExists
	}
	return id, err
}

func (r *Repository) Update(ctx context.Context, s domain.ShiftRow, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.shifts
		   SET shift_code = $2, shift_name = $3, start_time = $4::time, end_time = $5::time, sequence = $6,
		       is_active = $7, updated_by = $8
		 WHERE id = $1`, s.ID, s.Code, s.Name, s.Start, s.End, s.Sequence, s.IsActive, by)
	if database.IsUniqueViolation(err) {
		return apperror.ErrCodeExists
	}
	return err
}
