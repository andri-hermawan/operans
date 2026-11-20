// Package repository (fitur pagerule) = semua SQL ke tabel auth.access_rules.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/pagerule/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func scanRules(rows pgx.Rows) ([]domain.Rule, error) {
	defer rows.Close()
	out := []domain.Rule{}
	for rows.Next() {
		var r domain.Rule
		if err := rows.Scan(&r.SubjectType, &r.Role, &r.UserID, &r.PageKey, &r.Allowed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

const cols = `subject_type, role, user_id, page_key, allowed`

// ListProject = semua aturan satu project (untuk halaman pengaturan).
func (r *Repository) ListProject(ctx context.Context, projectID uuid.UUID) ([]domain.Rule, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM auth.access_rules WHERE project_id = $1 ORDER BY subject_type, role, user_id, page_key`, projectID)
	if err != nil {
		return nil, err
	}
	return scanRules(rows)
}

// ListFor = aturan yang berlaku untuk satu user: aturan role-nya + aturan khusus user itu.
func (r *Repository) ListFor(ctx context.Context, projectID uuid.UUID, role string, userID uuid.UUID) (roleRules, userRules []domain.Rule, err error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cols+` FROM auth.access_rules
		WHERE project_id = $1 AND ((subject_type = 'role' AND role = $2) OR (subject_type = 'user' AND user_id = $3))`, projectID, role, userID)
	if err != nil {
		return nil, nil, err
	}
	all, err := scanRules(rows)
	if err != nil {
		return nil, nil, err
	}
	for _, x := range all {
		if x.SubjectType == domain.SubjectRole {
			roleRules = append(roleRules, x)
		} else {
			userRules = append(userRules, x)
		}
	}
	return roleRules, userRules, nil
}

// IsMember = user terdaftar sebagai anggota project (aturan user hanya untuk anggota).
func (r *Repository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth.project_members WHERE project_id = $1 AND user_id = $2)`, projectID, userID).Scan(&ok)
	return ok, err
}

// Apply menyimpan sekumpulan perubahan untuk satu subjek dalam SATU transaksi (semua atau tidak sama sekali).
func (r *Repository) Apply(ctx context.Context, projectID uuid.UUID, in domain.SaveInput, by uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback setelah commit tidak berefek
	for _, c := range in.Changes {
		if err := applyOne(ctx, tx, projectID, in, c, by); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func applyOne(ctx context.Context, tx pgx.Tx, projectID uuid.UUID, in domain.SaveInput, c domain.Change, by uuid.UUID) error {
	if in.SubjectType == domain.SubjectRole {
		if c.Allowed == nil {
			_, err := tx.Exec(ctx, `DELETE FROM auth.access_rules WHERE project_id = $1 AND subject_type = 'role' AND role = $2 AND page_key = $3`, projectID, in.Role, c.PageKey)
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO auth.access_rules (project_id, subject_type, role, page_key, allowed, created_by, updated_by)
			VALUES ($1, 'role', $2, $3, $4, $5, $5)
			ON CONFLICT (project_id, role, page_key) WHERE subject_type = 'role'
			DO UPDATE SET allowed = EXCLUDED.allowed, updated_by = EXCLUDED.updated_by`, projectID, in.Role, c.PageKey, *c.Allowed, by)
		return err
	}
	if c.Allowed == nil {
		_, err := tx.Exec(ctx, `DELETE FROM auth.access_rules WHERE project_id = $1 AND subject_type = 'user' AND user_id = $2 AND page_key = $3`, projectID, in.UserID, c.PageKey)
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO auth.access_rules (project_id, subject_type, user_id, page_key, allowed, created_by, updated_by)
		VALUES ($1, 'user', $2, $3, $4, $5, $5)
		ON CONFLICT (project_id, user_id, page_key) WHERE subject_type = 'user'
		DO UPDATE SET allowed = EXCLUDED.allowed, updated_by = EXCLUDED.updated_by`, projectID, in.UserID, c.PageKey, *c.Allowed, by)
	return err
}
