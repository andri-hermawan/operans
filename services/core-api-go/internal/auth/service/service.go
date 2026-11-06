// Package service (fitur auth) = aturan login & sesi (ADR 0002): login, refresh token (rotation), pindah
// project, logout, profil /me, ganti password sendiri. Tidak tahu soal HTTP; database lewat interface.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/auth/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/token"
	"simha/core-api/pkg/hash"
)

// Repository = kemampuan database yang dibutuhkan login (dipenuhi *repository.Repository; test memakai versi palsu).
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
	SetLoginFailures(ctx context.Context, id uuid.UUID, count int, lockedUntil *time.Time) error
	RecordLoginSuccess(ctx context.Context, id uuid.UUID, newHash *string) error
	ListMemberships(ctx context.Context, userID uuid.UUID) ([]access.Membership, error)
	InsertRefreshToken(ctx context.Context, t domain.RefreshToken) error
	FindRefreshToken(ctx context.Context, hash string) (*domain.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldID uuid.UUID, next domain.RefreshToken) (bool, error)
	RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string) error
	SetPasswordByID(ctx context.Context, id uuid.UUID, hash string) error // juga mencabut semua sesi
	RevokeUserSessions(ctx context.Context, id uuid.UUID, reason string) error
}

// Policy = aturan sesi (ADR 0002): access 15 menit, refresh 7 hari tidak dipakai / maks. 30 hari.
type Policy struct {
	RefreshIdleTTL  time.Duration
	RefreshMaxTTL   time.Duration
	MaxFailedLogins int           // gagal berturut-turut sebelum akun dikunci
	LockDuration    time.Duration // lama akun dikunci
}

// Kesalahan yang dibedakan oleh lapisan HTTP. Pesannya sengaja umum agar tidak membocorkan info.
var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrNoProject          = errors.New("akun belum terdaftar di project mana pun; hubungi admin")
	ErrProjectForbidden   = errors.New("anda tidak terdaftar di project tersebut")
	ErrInvalidRefresh     = apperror.ErrInvalidRefresh // sama dengan galat bersama (response memetakannya ke 401)
	ErrRefreshReused      = errors.New("sesi dihentikan demi keamanan (token dipakai ulang), silakan login ulang")
	ErrWrongPassword      = errors.New("password saat ini salah")
)

// LockedError = akun sedang dikunci karena terlalu banyak login gagal.
type LockedError struct{ Until time.Time }

func (e *LockedError) Error() string {
	return fmt.Sprintf("terlalu banyak percobaan login gagal, coba lagi setelah %s", e.Until.Format("15:04"))
}

// Client = informasi perangkat peminta (disimpan bersama refresh token untuk audit).
type Client struct {
	UserAgent string
	IP        string
}

// Session = hasil login / refresh yang dikirim ke frontend.
type Session struct {
	AccessToken      string             `json:"access_token"`
	TokenType        string             `json:"token_type"`
	ExpiresIn        int                `json:"expires_in"` // detik
	AccessExpiresAt  time.Time          `json:"access_expires_at"`
	RefreshToken     string             `json:"refresh_token"`
	RefreshExpiresAt time.Time          `json:"refresh_expires_at"`
	User             SessionUser        `json:"user"`
	Project          *access.Membership `json:"project"` // project aktif; null = superadmin tanpa project
}

type SessionUser struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	IsSuperadmin bool      `json:"is_superadmin"`
}

// ProjectFinder = fitur project (superadmin boleh membuka project aktif mana pun).
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

type Service struct {
	repo     Repository
	projects ProjectFinder
	signer   *token.Signer
	policy   Policy
	now      func() time.Time
}

func New(repo Repository, projects ProjectFinder, signer *token.Signer, policy Policy) *Service {
	return &Service{repo: repo, projects: projects, signer: signer, policy: policy, now: time.Now}
}

// ---------------------------------------------------------------------
// Dipakai fitur user (kelola user) — akun & sesi tetap diurus fitur auth
// ---------------------------------------------------------------------

// FindUserByEmail: (nil, nil) bila tidak ada. Email disimpan huruf kecil.
func (s *Service) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.FindUserByEmail(ctx, email)
}

// SetPasswordByID mengganti password (hash), membuka kunci akun, dan mencabut SEMUA sesi user.
func (s *Service) SetPasswordByID(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return s.repo.SetPasswordByID(ctx, id, passwordHash)
}

// RevokeUserSessions mencabut semua refresh token user (mis. saat dinonaktifkan).
func (s *Service) RevokeUserSessions(ctx context.Context, id uuid.UUID, reason string) error {
	return s.repo.RevokeUserSessions(ctx, id, reason)
}

// Login: email + password → sesi baru. projectID opsional (project yang ingin dibuka).
func (s *Service) Login(ctx context.Context, email, password string, projectID *uuid.UUID, client Client) (*Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		hash.DummyVerify(password) // jawaban sama lamanya: tidak membocorkan email mana yang terdaftar
		return nil, ErrInvalidCredentials
	}
	now := s.now()
	if u.LockedUntil != nil && now.Before(*u.LockedUntil) {
		return nil, &LockedError{Until: *u.LockedUntil}
	}

	ok, needsRehash := hash.VerifyPassword(u.PasswordHash, password)
	if !ok {
		count, lock := u.FailedLoginCount+1, (*time.Time)(nil)
		if count >= s.policy.MaxFailedLogins {
			until := now.Add(s.policy.LockDuration)
			count, lock = 0, &until
			slog.Warn("akun dikunci karena login gagal berulang", "user_id", u.ID, "sampai", until)
		}
		if err := s.repo.SetLoginFailures(ctx, u.ID, count, lock); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}

	var newHash *string
	if needsRehash { // hash bcrypt dari sistem lama → simpan ulang sebagai argon2id
		if h, err := hash.HashPassword(password); err == nil {
			newHash = &h
		}
	}
	if err := s.repo.RecordLoginSuccess(ctx, u.ID, newHash); err != nil {
		return nil, err
	}

	project, err := s.chooseProject(ctx, u, projectID, nil)
	if err != nil {
		return nil, err
	}
	refresh := domain.RefreshToken{
		ID: uuid.New(), FamilyID: uuid.New(), UserID: u.ID, IssuedAt: now,
		AbsoluteExpiresAt: now.Add(s.policy.RefreshMaxTTL),
		UserAgent:         truncate(client.UserAgent, 255), IPAddress: truncate(client.IP, 45),
	}
	return s.issue(ctx, u, project, refresh, func(t domain.RefreshToken) error { return s.repo.InsertRefreshToken(ctx, t) })
}

// Refresh menukar refresh token dengan sesi baru (token lama langsung tidak berlaku = rotation).
// projectID diisi untuk pindah project (switch-project); nil = tetap di project yang sama.
func (s *Service) Refresh(ctx context.Context, rawRefresh string, projectID *uuid.UUID, client Client) (*Session, error) {
	old, err := s.repo.FindRefreshToken(ctx, token.HashToken(rawRefresh))
	if err != nil {
		return nil, err
	}
	if old == nil || old.RevokedAt != nil {
		return nil, ErrInvalidRefresh
	}
	if old.UsedAt != nil {
		return nil, s.reuseDetected(ctx, old)
	}
	now := s.now()
	if now.After(old.ExpiresAt) || now.After(old.AbsoluteExpiresAt) {
		return nil, ErrInvalidRefresh
	}

	u, err := s.repo.GetUser(ctx, old.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		_ = s.repo.RevokeFamily(ctx, old.FamilyID, "user_inactive")
		return nil, ErrInvalidRefresh
	}
	project, err := s.chooseProject(ctx, u, projectID, old.ProjectID)
	if err != nil {
		return nil, err
	}

	next := domain.RefreshToken{
		ID: uuid.New(), FamilyID: old.FamilyID, UserID: u.ID, IssuedAt: now,
		AbsoluteExpiresAt: old.AbsoluteExpiresAt, // batas 30 hari tidak bertambah walau terus diperbarui
		UserAgent:         truncate(client.UserAgent, 255), IPAddress: truncate(client.IP, 45),
	}
	var reused bool
	session, err := s.issue(ctx, u, project, next, func(t domain.RefreshToken) error {
		rotated, err := s.repo.RotateRefreshToken(ctx, old.ID, t)
		reused = err == nil && !rotated
		return err
	})
	if reused { // token yang sama dipakai dua kali secara bersamaan
		return nil, s.reuseDetected(ctx, old)
	}
	return session, err
}

// Logout mencabut seluruh sesi milik refresh token itu. Selalu berhasil (idempotent).
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	t, err := s.repo.FindRefreshToken(ctx, token.HashToken(rawRefresh))
	if err != nil || t == nil {
		return err
	}
	return s.repo.RevokeFamily(ctx, t.FamilyID, "logout")
}

// ChangePassword: user mengganti password SENDIRI. Semua sesi dicabut → login ulang di semua perangkat
// (perangkat yang mungkin dipakai orang lain ikut keluar).
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword string) error {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil || !u.IsActive {
		return ErrInvalidRefresh
	}
	if ok, _ := hash.VerifyPassword(u.PasswordHash, current); !ok {
		return ErrWrongPassword
	}
	if err := hash.ValidatePassword(newPassword); err != nil {
		return &apperror.ValidationError{Msg: err.Error()}
	}
	if newPassword == current {
		return &apperror.ValidationError{Msg: "password baru harus berbeda dari password saat ini"}
	}
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.SetPasswordByID(ctx, u.ID, hashed)
}

// Profile = data untuk GET /auth/me.
type Profile struct {
	User          SessionUser         `json:"user"`
	ActiveProject *uuid.UUID          `json:"active_project_id"`
	Memberships   []access.Membership `json:"memberships"`
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID, activeProject *uuid.UUID) (*Profile, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return nil, ErrInvalidRefresh
	}
	ms, err := s.repo.ListMemberships(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return &Profile{User: sessionUser(u), ActiveProject: activeProject, Memberships: ms}, nil
}

// chooseProject menentukan project aktif sesi:
//  1. project yang diminta (harus anggota; superadmin boleh project aktif mana pun)
//  2. project sesi sebelumnya (bila masih anggota)
//  3. project pertama tempat user menjadi anggota
//  4. superadmin tanpa keanggotaan → tanpa project (akses lintas project)
func (s *Service) chooseProject(ctx context.Context, u *domain.User, requested, previous *uuid.UUID) (*access.Membership, error) {
	ms, err := s.repo.ListMemberships(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	find := func(id uuid.UUID) *access.Membership {
		for i := range ms {
			if ms[i].ProjectID == id {
				return &ms[i]
			}
		}
		return nil
	}

	if requested != nil {
		if m := find(*requested); m != nil {
			return m, nil
		}
		if u.IsSuperadmin {
			p, err := s.projects.FindActive(ctx, *requested)
			if err != nil {
				return nil, err
			}
			if p != nil {
				return &access.Membership{ProjectID: p.ID, ProjectCode: p.Code, ProjectName: p.Name}, nil
			}
		}
		return nil, ErrProjectForbidden
	}
	if previous != nil {
		if m := find(*previous); m != nil {
			return m, nil
		}
		if u.IsSuperadmin {
			return s.chooseProject(ctx, u, previous, nil)
		}
	}
	if len(ms) > 0 {
		return &ms[0], nil
	}
	if u.IsSuperadmin {
		return nil, nil
	}
	return nil, ErrNoProject
}

// issue membuat access token + refresh token baru, lalu menyimpan refresh token lewat save.
func (s *Service) issue(ctx context.Context, u *domain.User, project *access.Membership, refresh domain.RefreshToken,
	save func(domain.RefreshToken) error) (*Session, error) {

	sub := token.AccessSubject{UserID: u.ID, Name: u.Name, SuperAdmin: u.IsSuperadmin}
	if project != nil {
		sub.ProjectID, sub.ProjectRole = &project.ProjectID, project.Role
		refresh.ProjectID = &project.ProjectID
	}
	access, accessExp, err := s.signer.IssueAccess(sub)
	if err != nil {
		return nil, err
	}

	raw, tokenHash, err := token.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	refresh.TokenHash = tokenHash
	refresh.ExpiresAt = minTime(refresh.IssuedAt.Add(s.policy.RefreshIdleTTL), refresh.AbsoluteExpiresAt)
	if err := save(refresh); err != nil {
		return nil, err
	}

	return &Session{
		AccessToken: access, TokenType: "Bearer", ExpiresIn: int(time.Until(accessExp).Seconds()),
		AccessExpiresAt: accessExp, RefreshToken: raw, RefreshExpiresAt: refresh.ExpiresAt,
		User: sessionUser(u), Project: project,
	}, nil
}

func (s *Service) reuseDetected(ctx context.Context, t *domain.RefreshToken) error {
	slog.Warn("refresh token dipakai ulang: seluruh sesi dicabut", "user_id", t.UserID, "family_id", t.FamilyID)
	if err := s.repo.RevokeFamily(ctx, t.FamilyID, "reuse_detected"); err != nil {
		return err
	}
	return ErrRefreshReused
}

func sessionUser(u *domain.User) SessionUser {
	return SessionUser{ID: u.ID, Email: u.Email, Name: u.Name, IsSuperadmin: u.IsSuperadmin}
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
