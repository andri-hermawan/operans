// Package repository (fitur project) = semua SQL ke tabel core.projects.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// GetActive: (nil, nil) bila tidak ada atau nonaktif.
func (r *Repository) GetActive(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return scanProject(r.pool.QueryRow(ctx,
		`SELECT id, project_code, project_name, timezone FROM core.projects WHERE id = $1 AND is_active`, id))
}

// GetByCode: (nil, nil) bila tidak ada atau nonaktif (dipakai tools/create-user).
func (r *Repository) GetByCode(ctx context.Context, code string) (*domain.Project, error) {
	return scanProject(r.pool.QueryRow(ctx,
		`SELECT id, project_code, project_name, timezone FROM core.projects WHERE project_code = $1 AND is_active`, code))
}

func scanProject(row pgx.Row) (*domain.Project, error) {
	var p domain.Project
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

const projectCols = `p.id, p.company_id, c.company_name, p.project_code, p.project_name, p.timezone, p.is_active,
    p.created_at, p.updated_at`

func scanProjectDetail(row pgx.Row) (*domain.ProjectDetail, error) {
	var p domain.ProjectDetail
	err := row.Scan(&p.ID, &p.CompanyID, &p.CompanyName, &p.ProjectCode, &p.ProjectName, &p.Timezone, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// List: ids nil = semua project.
func (r *Repository) List(ctx context.Context, ids []uuid.UUID) ([]domain.ProjectDetail, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+projectCols+`
		  FROM core.projects p JOIN core.companies c ON c.id = p.company_id
		 WHERE $1::uuid[] IS NULL OR p.id = ANY($1)
		 ORDER BY p.project_code`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ProjectDetail{}
	for rows.Next() {
		p, err := scanProjectDetail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) GetDetail(ctx context.Context, id uuid.UUID) (*domain.ProjectDetail, error) {
	return scanProjectDetail(r.pool.QueryRow(ctx, `SELECT `+projectCols+`
		  FROM core.projects p JOIN core.companies c ON c.id = p.company_id WHERE p.id = $1`, id))
}

func (r *Repository) Insert(ctx context.Context, p domain.ProjectDetail, by uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.projects (company_id, project_code, project_name, timezone, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6) RETURNING id`,
		p.CompanyID, p.ProjectCode, p.ProjectName, p.Timezone, p.IsActive, by).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, apperror.ErrCodeExists
	}
	return id, err
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, in domain.ProjectInput, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.projects
		   SET company_id = COALESCE($2, company_id), project_code = COALESCE($3, project_code),
		       project_name = COALESCE($4, project_name), timezone = COALESCE($5, timezone),
		       is_active = COALESCE($6, is_active), updated_by = $7
		 WHERE id = $1`, id, in.CompanyID, in.ProjectCode, in.ProjectName, in.Timezone, in.IsActive, by)
	if database.IsUniqueViolation(err) {
		return apperror.ErrCodeExists
	}
	return err
}
