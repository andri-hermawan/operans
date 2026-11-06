// Package repository (fitur equipment) = semua SQL ke tabel core.equipment.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/equipment/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const equipmentCols = `id, project_id, equipment_code, equipment_alias, equipment_type, brand, model, class,
    is_active, created_at, updated_at`

func scanEquipment(row pgx.Row) (*domain.Equipment, error) {
	var e domain.Equipment
	err := row.Scan(&e.ID, &e.ProjectID, &e.EquipmentCode, &e.EquipmentAlias, &e.EquipmentType, &e.Brand,
		&e.Model, &e.Class, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// List: projectID nil = semua project; q = cari di kode/alias. Maks. 2000 baris.
func (r *Repository) List(ctx context.Context, projectID *uuid.UUID, q string, includeInactive bool) ([]domain.Equipment, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+equipmentCols+` FROM core.equipment
		 WHERE ($1::uuid IS NULL OR project_id = $1)
		   AND ($2 OR is_active)
		   AND ($3::text = '' OR equipment_code ILIKE '%' || $3 || '%' OR equipment_alias ILIKE '%' || $3 || '%')
		 ORDER BY equipment_code
		 LIMIT 2000`, projectID, includeInactive, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Equipment{}
	for rows.Next() {
		e, err := scanEquipment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*domain.Equipment, error) {
	return scanEquipment(r.pool.QueryRow(ctx, `SELECT `+equipmentCols+` FROM core.equipment WHERE id = $1`, id))
}

func (r *Repository) Insert(ctx context.Context, e domain.Equipment, by uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO core.equipment (project_id, equipment_code, equipment_alias, equipment_type, brand, model, class,
		                            is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		RETURNING id`,
		e.ProjectID, e.EquipmentCode, e.EquipmentAlias, e.EquipmentType, e.Brand, e.Model, e.Class, e.IsActive, by).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, apperror.ErrCodeExists
	}
	return id, err
}

// Update mengubah kolom yang tidak nil.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, in domain.EquipmentInput, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE core.equipment
		   SET equipment_code  = COALESCE($2, equipment_code),
		       equipment_alias = COALESCE($3, equipment_alias),
		       equipment_type  = COALESCE($4, equipment_type),
		       brand           = COALESCE($5, brand),
		       model           = COALESCE($6, model),
		       class           = COALESCE($7, class),
		       is_active       = COALESCE($8, is_active),
		       updated_by      = $9
		 WHERE id = $1`,
		id, in.EquipmentCode, in.EquipmentAlias, in.EquipmentType, in.Brand, in.Model, in.Class, in.IsActive, by)
	if database.IsUniqueViolation(err) {
		return apperror.ErrCodeExists
	}
	return err
}

// CodeMap = kode → id semua equipment project (untuk menautkan kode unit di data harian).
func (r *Repository) CodeMap(ctx context.Context, projectID uuid.UUID) (map[string]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT equipment_code, id FROM core.equipment WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]uuid.UUID{}
	for rows.Next() {
		var code string
		var id uuid.UUID
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		out[code] = id
	}
	return out, rows.Err()
}
