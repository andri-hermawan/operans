// Package repository (fitur user) = SQL halaman kelola user: daftar & ringkasan user (auth.users) beserta
// keanggotaan project (auth.project_members). SQL login & sesi ada di fitur auth.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "simha/core-api/internal/auth/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/user/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// summarySQL: user + daftar keanggotaan (project aktif saja) dalam SATU query.
const summarySQL = `
SELECT u.id, u.email, u.name, u.phone_number, u.is_superadmin, u.is_active, u.last_login_at, u.created_at,
       COALESCE(json_agg(json_build_object(
                    'project_id', p.id, 'project_code', p.project_code,
                    'project_name', p.project_name, 'role', m.role)
                ORDER BY p.project_code) FILTER (WHERE p.id IS NOT NULL), '[]')
  FROM auth.users u
  LEFT JOIN auth.project_members m ON m.user_id = u.id
  LEFT JOIN core.projects p        ON p.id = m.project_id AND p.is_active`

func scanSummary(row pgx.Row) (*domain.UserSummary, error) {
	var u domain.UserSummary
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PhoneNumber, &u.IsSuperadmin, &u.IsActive, &u.LastLoginAt,
		&u.CreatedAt, &u.Memberships)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// List: projectID nil = semua user; q = cari di email/nama. Maks. 500 baris.
func (r *Repository) List(ctx context.Context, projectID *uuid.UUID, q string) ([]domain.UserSummary, error) {
	rows, err := r.pool.Query(ctx, summarySQL+`
		 WHERE ($1::uuid IS NULL OR EXISTS (
		           SELECT 1 FROM auth.project_members x WHERE x.user_id = u.id AND x.project_id = $1))
		   AND ($2 = '' OR u.email ILIKE '%' || $2 || '%' OR u.name ILIKE '%' || $2 || '%')
		 GROUP BY u.id
		 ORDER BY u.name
		 LIMIT 500`, projectID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UserSummary{}
	for rows.Next() {
		u, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// GetSummary: (nil, nil) bila tidak ada.
func (r *Repository) GetSummary(ctx context.Context, id uuid.UUID) (*domain.UserSummary, error) {
	return scanSummary(r.pool.QueryRow(ctx, summarySQL+` WHERE u.id = $1 GROUP BY u.id`, id))
}

// Insert membuat user (+ keanggotaan project bila ada) dalam SATU transaksi.
func (r *Repository) Insert(ctx context.Context, u authdomain.User, by uuid.UUID, projectID *uuid.UUID, role string) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO auth.users (email, name, phone_number, password_hash, is_superadmin, password_changed_at,
		                        created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, now(), $6, $6)
		RETURNING id`, u.Email, u.Name, u.PhoneNumber, u.PasswordHash, u.IsSuperadmin, by).Scan(&id)
	if database.IsUniqueViolation(err) {
		return uuid.Nil, apperror.ErrEmailExists
	}
	if err != nil {
		return uuid.Nil, err
	}
	if projectID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth.project_members (user_id, project_id, role, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $4)`, id, *projectID, role, by); err != nil {
			return uuid.Nil, err
		}
	}
	return id, tx.Commit(ctx)
}

// Update mengubah kolom yang tidak nil. PhoneNumber "" = dikosongkan.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, upd domain.UserUpdate, by uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE auth.users
		   SET name          = COALESCE($2, name),
		       phone_number  = CASE WHEN $3::text IS NULL THEN phone_number ELSE NULLIF($3, '') END,
		       is_active     = COALESCE($4, is_active),
		       is_superadmin = COALESCE($5, is_superadmin),
		       updated_by    = $6
		 WHERE id = $1`, id, upd.Name, upd.PhoneNumber, upd.IsActive, upd.IsSuperadmin, by)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// RemoveMembership: removed=false bila user memang bukan anggota project itu.
func (r *Repository) RemoveMembership(ctx context.Context, userID, projectID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM auth.project_members WHERE user_id = $1 AND project_id = $2`, userID, projectID)
	return tag.RowsAffected() == 1, err
}

// SetMembership menambah atau mengubah role user di satu project. by = pelaku (nil: alat baris perintah).
func (r *Repository) SetMembership(ctx context.Context, userID, projectID uuid.UUID, role string, by *uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth.project_members (user_id, project_id, role, created_by, updated_by) VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (user_id, project_id) DO UPDATE SET role = EXCLUDED.role, updated_by = EXCLUDED.updated_by`,
		userID, projectID, role, by)
	return err
}

// CreateUser membuat user baru (dipakai tools/create-user; kelola user lengkap di B3).
func (r *Repository) CreateUser(ctx context.Context, u authdomain.User) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO auth.users (email, name, phone_number, password_hash, is_superadmin, password_changed_at)
		VALUES ($1, $2, $3, $4, $5, now())
		RETURNING id`, u.Email, u.Name, u.PhoneNumber, u.PasswordHash, u.IsSuperadmin).Scan(&id)
	return id, err
}
