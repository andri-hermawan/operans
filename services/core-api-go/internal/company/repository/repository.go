// Package repository (fitur company) = semua SQL ke tabel core.companies.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/company/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// List: ids nil = semua company; selain itu hanya company dari project tersebut.
func (r *Repository) List(ctx context.Context, projectIDs []uuid.UUID) ([]domain.Company, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.company_code, c.company_name, c.is_active,
		       (SELECT count(*) FROM core.projects p WHERE p.company_id = c.id), c.created_at, c.updated_at
		  FROM core.companies c
		 WHERE $1::uuid[] IS NULL OR c.id IN (SELECT company_id FROM core.projects WHERE id = ANY($1))
		 ORDER BY c.company_code`, projectIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Company{}
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(&c.ID, &c.CompanyCode, &c.CompanyName, &c.IsActive, &c.ProjectCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Insert(ctx context.Context, code, name string, active bool, by uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.companies (company_code, company_name, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $4) RETURNING id`, code, name, active, by).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, apperror.ErrCodeExists
	}
	return id, err
}

// Update mengubah kolom yang tidak nil; false bila company tidak ada.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, in domain.CompanyInput, by uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE core.companies
		   SET company_code = COALESCE($2, company_code), company_name = COALESCE($3, company_name),
		       is_active = COALESCE($4, is_active), updated_by = $5
		 WHERE id = $1`, id, in.CompanyCode, in.CompanyName, in.IsActive, by)
	if database.IsUniqueViolation(err) {
		return false, apperror.ErrCodeExists
	}
	return tag.RowsAffected() == 1, err
}

func (r *Repository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM core.companies WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}
