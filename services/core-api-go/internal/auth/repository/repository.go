// Package repository (fitur auth) = SQL untuk login & sesi: membaca akun user (auth.users) saat login,
// mencatat login gagal/berhasil, keanggotaan project user, refresh token (auth.refresh_tokens),
// serta ganti password + cabut sesi.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/auth/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ---------------------------------------------------------------------
// Akun user untuk login (dulu repository/users.go)
// ---------------------------------------------------------------------

const userCols = `id, email, name, phone_number, password_hash, is_superadmin, is_active,
    failed_login_count, locked_until, last_login_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PhoneNumber, &u.PasswordHash, &u.IsSuperadmin, &u.IsActive,
		&u.FailedLoginCount, &u.LockedUntil, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindUserByEmail: (nil, nil) bila tidak ada. Email disimpan huruf kecil (CHECK di database).
func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM auth.users WHERE email = $1`, email))
}

func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM auth.users WHERE id = $1`, id))
}

// SetLoginFailures mencatat jumlah login gagal dan (bila perlu) mengunci akun sampai lockedUntil.
func (r *Repository) SetLoginFailures(ctx context.Context, id uuid.UUID, count int, lockedUntil *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE auth.users SET failed_login_count = $2, locked_until = $3 WHERE id = $1`, id, count, lockedUntil)
	return err
}

// RecordLoginSuccess mereset hitungan gagal; newHash diisi bila hash lama (bcrypt) perlu diganti argon2id.
func (r *Repository) RecordLoginSuccess(ctx context.Context, id uuid.UUID, newHash *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth.users
		   SET failed_login_count = 0, locked_until = NULL, last_login_at = now(),
		       password_hash = COALESCE($2, password_hash)
		 WHERE id = $1`, id, newHash)
	return err
}

// ResetPassword (dipakai tools/create-user) mengganti password user berdasarkan email, membuka kunci akun, dan mencabut semua sesinya.
// found=false bila email tidak terdaftar.
func (r *Repository) ResetPassword(ctx context.Context, email, hash string) (found bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE auth.users
		   SET password_hash = $2, password_changed_at = now(), failed_login_count = 0, locked_until = NULL
		 WHERE email = $1
		RETURNING id`, email, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth.refresh_tokens SET revoked_at = now(), revoked_reason = 'password_changed'
		 WHERE user_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// ListMemberships = semua project AKTIF tempat user menjadi anggota, urut kode project.
func (r *Repository) ListMemberships(ctx context.Context, userID uuid.UUID) ([]access.Membership, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.project_code, p.project_name, m.role
		  FROM auth.project_members m
		  JOIN core.projects p ON p.id = m.project_id
		 WHERE m.user_id = $1 AND p.is_active
		 ORDER BY p.project_code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []access.Membership{}
	for rows.Next() {
		var m access.Membership
		if err := rows.Scan(&m.ProjectID, &m.ProjectCode, &m.ProjectName, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------
// Ganti password & cabut sesi (dulu repository/users_admin.go; dipakai juga fitur user lewat service auth)
// ---------------------------------------------------------------------

// SetPasswordByID mengganti password, membuka kunci akun, dan mencabut SEMUA sesi user (satu transaksi).
func (r *Repository) SetPasswordByID(ctx context.Context, id uuid.UUID, hash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE auth.users
		   SET password_hash = $2, password_changed_at = now(), failed_login_count = 0, locked_until = NULL
		 WHERE id = $1`, id, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth.refresh_tokens SET revoked_at = now(), revoked_reason = 'password_changed'
		 WHERE user_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RevokeUserSessions mencabut semua refresh token user (mis. saat dinonaktifkan).
func (r *Repository) RevokeUserSessions(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth.refresh_tokens SET revoked_at = now(), revoked_reason = $2
		 WHERE user_id = $1 AND revoked_at IS NULL`, id, reason)
	return err
}

// ---------------------------------------------------------------------
// Refresh token (dulu repository/refresh_tokens.go)
// ---------------------------------------------------------------------

func (r *Repository) InsertRefreshToken(ctx context.Context, t domain.RefreshToken) error {
	_, err := r.pool.Exec(ctx, insertRefreshSQL, refreshArgs(t)...)
	return err
}

const insertRefreshSQL = `
INSERT INTO auth.refresh_tokens (id, family_id, user_id, project_id, token_hash, issued_at,
                                 expires_at, absolute_expires_at, user_agent, ip_address)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

func refreshArgs(t domain.RefreshToken) []any {
	return []any{t.ID, t.FamilyID, t.UserID, t.ProjectID, t.TokenHash, t.IssuedAt,
		t.ExpiresAt, t.AbsoluteExpiresAt, t.UserAgent, t.IPAddress}
}

// FindRefreshToken mencari berdasarkan hash. (nil, nil) bila tidak ada.
func (r *Repository) FindRefreshToken(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, family_id, user_id, project_id, token_hash, issued_at, expires_at, absolute_expires_at,
		       used_at, revoked_at
		  FROM auth.refresh_tokens WHERE token_hash = $1`, hash).Scan(
		&t.ID, &t.FamilyID, &t.UserID, &t.ProjectID, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.AbsoluteExpiresAt,
		&t.UsedAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// RotateRefreshToken menandai token lama "sudah dipakai" dan menyimpan penggantinya dalam SATU transaksi.
// rotated=false berarti token lama ternyata sudah dipakai/dicabut lebih dulu (mis. dua permintaan
// bersamaan dengan token yang sama) — pemanggil harus menganggapnya pemakaian ulang.
func (r *Repository) RotateRefreshToken(ctx context.Context, oldID uuid.UUID, next domain.RefreshToken) (rotated bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) // tidak berpengaruh bila Commit sudah berhasil

	tag, err := tx.Exec(ctx, `
		UPDATE auth.refresh_tokens SET used_at = now(), replaced_by = $2
		 WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL`, oldID, next.ID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() != 1 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, insertRefreshSQL, refreshArgs(next)...); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// RevokeFamily mencabut SEMUA token dalam satu sesi login (logout, atau pencurian terdeteksi).
func (r *Repository) RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE auth.refresh_tokens SET revoked_at = now(), revoked_reason = $2
		 WHERE family_id = $1 AND revoked_at IS NULL`, familyID, reason)
	return err
}
