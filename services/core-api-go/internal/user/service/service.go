// Package service (fitur user) = aturan kelola user & keanggotaan project (B3):
// superadmin boleh semua; admin project hanya user di project aktifnya.
// Akun (cari email, ganti password, cabut sesi) diurus fitur auth; project aktif dicek lewat fitur project.
package service

import (
	"context"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	authdomain "simha/core-api/internal/auth/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/user/domain"
	"simha/core-api/pkg/hash"
)

// Repository = kemampuan database untuk kelola user (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectID *uuid.UUID, q string) ([]domain.UserSummary, error)
	GetSummary(ctx context.Context, id uuid.UUID) (*domain.UserSummary, error)
	Insert(ctx context.Context, u authdomain.User, by uuid.UUID, projectID *uuid.UUID, role string) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, upd domain.UserUpdate, by uuid.UUID) error
	SetMembership(ctx context.Context, userID, projectID uuid.UUID, role string, by *uuid.UUID) error
	RemoveMembership(ctx context.Context, userID, projectID uuid.UUID) (bool, error)
}

// Accounts = fitur auth (akun & sesi login).
type Accounts interface {
	FindUserByEmail(ctx context.Context, email string) (*authdomain.User, error)
	SetPasswordByID(ctx context.Context, id uuid.UUID, hash string) error // juga mencabut semua sesi
	RevokeUserSessions(ctx context.Context, id uuid.UUID, reason string) error
}

// ProjectFinder = fitur project.
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

type Service struct {
	repo     Repository
	accounts Accounts
	projects ProjectFinder
}

func New(repo Repository, accounts Accounts, projects ProjectFinder) *Service {
	return &Service{repo: repo, accounts: accounts, projects: projects}
}

// List: superadmin → semua user (opsional filter project); admin → anggota project aktifnya saja.
func (s *Service) List(ctx context.Context, a access.Actor, projectFilter *uuid.UUID, q string) ([]domain.UserSummary, error) {
	scope, err := s.listScope(a, projectFilter)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.List(ctx, scope, strings.TrimSpace(q))
	if err != nil {
		return nil, err
	}
	if !a.SuperAdmin { // admin project hanya melihat keanggotaan di project-nya sendiri
		for i := range users {
			users[i].Memberships = onlyProject(users[i].Memberships, *a.ProjectID)
		}
	}
	return users, nil
}

func (s *Service) listScope(a access.Actor, filter *uuid.UUID) (*uuid.UUID, error) {
	if a.SuperAdmin {
		return filter, nil
	}
	if a.Role != access.RoleAdmin || a.ProjectID == nil {
		return nil, apperror.ErrForbidden
	}
	if filter != nil && *filter != *a.ProjectID {
		return nil, apperror.ErrForbidden
	}
	return a.ProjectID, nil
}

// Get: satu user, dengan aturan lihat yang sama seperti List.
func (s *Service) Get(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.UserSummary, error) {
	u, err := s.visibleUser(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !a.SuperAdmin {
		u.Memberships = onlyProject(u.Memberships, *a.ProjectID)
	}
	return u, nil
}

// NewUser = isian membuat user baru.
type NewUser struct {
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PhoneNumber  *string    `json:"phone_number"`
	Password     string     `json:"password"`
	IsSuperadmin bool       `json:"is_superadmin"` // hanya superadmin
	ProjectID    *uuid.UUID `json:"project_id"`    // admin: diabaikan, selalu project aktifnya
	Role         string     `json:"role"`
}

// Create: superadmin bebas; admin hanya membuat user di project aktifnya dan tidak bisa membuat superadmin.
func (s *Service) Create(ctx context.Context, a access.Actor, in NewUser) (*domain.UserSummary, error) {
	if !a.SuperAdmin {
		if a.Role != access.RoleAdmin || a.ProjectID == nil || in.IsSuperadmin {
			return nil, apperror.ErrForbidden
		}
		in.ProjectID = a.ProjectID
	}
	email, err := cleanEmail(in.Email)
	if err != nil {
		return nil, err
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return nil, err
	}
	phone, err := cleanPhone(in.PhoneNumber)
	if err != nil {
		return nil, err
	}
	if err := hash.ValidatePassword(in.Password); err != nil {
		return nil, &apperror.ValidationError{Msg: err.Error()}
	}
	if in.ProjectID != nil {
		if !access.ValidRole(in.Role) {
			return nil, apperror.Invalid("role harus admin, dispatcher, atau viewer")
		}
		if p, err := s.projects.FindActive(ctx, *in.ProjectID); err != nil {
			return nil, err
		} else if p == nil {
			return nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
		}
	} else if !in.IsSuperadmin {
		return nil, apperror.Invalid("user biasa wajib punya project & role")
	}

	hashed, err := hash.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.Insert(ctx, authdomain.User{
		Email: email, Name: name, PhoneNumber: phone, PasswordHash: hashed, IsSuperadmin: in.IsSuperadmin,
	}, a.UserID, in.ProjectID, in.Role)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, a, id)
}

// Update: nama & nomor HP oleh superadmin atau admin project user itu.
// Status aktif & superadmin HANYA oleh superadmin, dan tidak untuk diri sendiri.
func (s *Service) Update(ctx context.Context, a access.Actor, id uuid.UUID, upd domain.UserUpdate) (*domain.UserSummary, error) {
	target, err := s.visibleUser(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if upd.IsActive != nil || upd.IsSuperadmin != nil {
		if !a.SuperAdmin {
			return nil, apperror.ErrForbidden
		}
		if id == a.UserID {
			return nil, apperror.ErrSelf
		}
	}
	if !a.SuperAdmin && target.IsSuperadmin {
		return nil, apperror.ErrForbidden // admin project tidak boleh mengubah data superadmin
	}
	if upd.Name != nil {
		n, err := cleanName(*upd.Name)
		if err != nil {
			return nil, err
		}
		upd.Name = &n
	}
	if upd.PhoneNumber != nil {
		p, err := cleanPhone(upd.PhoneNumber)
		if err != nil {
			return nil, err
		}
		empty := ""
		if p == nil {
			p = &empty // "" → repository mengosongkan kolom
		}
		upd.PhoneNumber = p
	}
	if err := s.repo.Update(ctx, id, upd, a.UserID); err != nil {
		return nil, err
	}
	if upd.IsActive != nil && !*upd.IsActive { // dinonaktifkan → semua sesinya langsung berakhir
		if err := s.accounts.RevokeUserSessions(ctx, id, "deactivated"); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, a, id)
}

// ResetPassword oleh superadmin: password baru + semua sesi user dicabut.
func (s *Service) ResetPassword(ctx context.Context, a access.Actor, id uuid.UUID, newPassword string) error {
	if !a.SuperAdmin {
		return apperror.ErrForbidden
	}
	if u, err := s.repo.GetSummary(ctx, id); err != nil {
		return err
	} else if u == nil {
		return apperror.ErrNotFound
	}
	if err := hash.ValidatePassword(newPassword); err != nil {
		return &apperror.ValidationError{Msg: err.Error()}
	}
	hashed, err := hash.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.accounts.SetPasswordByID(ctx, id, hashed)
}

// SetMember menambah user (berdasarkan email) ke project, atau mengubah role-nya.
func (s *Service) SetMember(ctx context.Context, a access.Actor, projectID uuid.UUID, email, role string) (*domain.UserSummary, error) {
	if !a.CanManageProject(projectID) {
		return nil, apperror.ErrForbidden
	}
	if !access.ValidRole(role) {
		return nil, apperror.Invalid("role harus admin, dispatcher, atau viewer")
	}
	email, err := cleanEmail(email)
	if err != nil {
		return nil, err
	}
	u, err := s.accounts.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return nil, apperror.ErrNotFound
	}
	if u.ID == a.UserID {
		return nil, apperror.ErrSelf // mencegah admin menurunkan role-nya sendiri lalu terkunci
	}
	if p, err := s.projects.FindActive(ctx, projectID); err != nil {
		return nil, err
	} else if p == nil {
		return nil, apperror.ErrNotFound
	}
	if err := s.repo.SetMembership(ctx, u.ID, projectID, role, &a.UserID); err != nil {
		return nil, err
	}
	sum, err := s.repo.GetSummary(ctx, u.ID)
	if err != nil || sum == nil {
		return nil, err
	}
	if !a.SuperAdmin {
		sum.Memberships = onlyProject(sum.Memberships, projectID)
	}
	return sum, nil
}

// RemoveMember mengeluarkan user dari project (akun tetap ada).
func (s *Service) RemoveMember(ctx context.Context, a access.Actor, projectID, userID uuid.UUID) error {
	if !a.CanManageProject(projectID) {
		return apperror.ErrForbidden
	}
	if userID == a.UserID {
		return apperror.ErrSelf
	}
	removed, err := s.repo.RemoveMembership(ctx, userID, projectID)
	if err != nil {
		return err
	}
	if !removed {
		return apperror.ErrNotFound
	}
	return nil
}

// visibleUser: superadmin melihat semua user; admin hanya anggota project aktifnya.
func (s *Service) visibleUser(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.UserSummary, error) {
	if !a.SuperAdmin && (a.Role != access.RoleAdmin || a.ProjectID == nil) {
		return nil, apperror.ErrForbidden
	}
	u, err := s.repo.GetSummary(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil || (!a.SuperAdmin && len(onlyProject(u.Memberships, *a.ProjectID)) == 0) {
		return nil, apperror.ErrNotFound // user project lain = "tidak ditemukan", tidak membocorkan keberadaannya
	}
	return u, nil
}

func onlyProject(ms []access.Membership, p uuid.UUID) []access.Membership {
	out := []access.Membership{}
	for _, m := range ms {
		if m.ProjectID == p {
			out = append(out, m)
		}
	}
	return out
}

func cleanEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 150 {
		return "", apperror.Invalid("format email tidak valid")
	}
	return s, nil
}

func cleanName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 150 {
		return "", apperror.Invalid("nama wajib diisi (maks. 150 karakter)")
	}
	return s, nil
}

// cleanPhone: nil/"" = tidak ada; selain itu hanya angka (boleh diawali +), 8–20 karakter.
func cleanPhone(p *string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	s := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(*p), " ", ""), "-", "")
	if s == "" {
		return nil, nil
	}
	digits := strings.TrimPrefix(s, "+")
	if len(s) < 8 || len(s) > 20 || strings.Trim(digits, "0123456789") != "" {
		return nil, apperror.Invalid("nomor HP hanya angka (boleh diawali +), 8–20 karakter")
	}
	return &s, nil
}
