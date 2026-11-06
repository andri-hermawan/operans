// Package access = SIAPA yang memanggil API (Actor, dari access token) dan aturan hak aksesnya:
// role per project, boleh melihat / mengelola / mengisi data harian. Dipakai service semua fitur.
package access

import (
	"github.com/google/uuid"

	"simha/core-api/internal/shared/apperror"
)

// Role per project (ADR 0002). Superadmin bukan role project, melainkan tanda di User.
const (
	RoleAdmin      = "admin"
	RoleDispatcher = "dispatcher"
	RoleViewer     = "viewer"
)

func ValidRole(r string) bool { return r == RoleAdmin || r == RoleDispatcher || r == RoleViewer }

// Membership = keanggotaan user di satu project beserta role-nya (dipakai fitur auth & user).
type Membership struct {
	ProjectID   uuid.UUID `json:"project_id"`
	ProjectCode string    `json:"project_code"`
	ProjectName string    `json:"project_name"`
	Role        string    `json:"role"`
}

// Actor = SIAPA yang sedang memanggil API (dari access token).
type Actor struct {
	UserID     uuid.UUID
	ProjectID  *uuid.UUID // project aktif
	Role       string     // role di project aktif
	SuperAdmin bool
}

// CanManageProject: superadmin boleh semua project; admin hanya project aktifnya (data induk, user).
func (a Actor) CanManageProject(projectID uuid.UUID) bool {
	return a.SuperAdmin || (a.Role == RoleAdmin && a.inProject(projectID))
}

// CanOperate: input operasional harian (aktivasi, operator, breakdown, …) — admin & dispatcher.
func (a Actor) CanOperate(projectID uuid.UUID) bool {
	return a.SuperAdmin || ((a.Role == RoleAdmin || a.Role == RoleDispatcher) && a.inProject(projectID))
}

// CanView: semua role di project aktifnya boleh melihat.
func (a Actor) CanView(projectID uuid.UUID) bool {
	return a.SuperAdmin || a.inProject(projectID)
}

// ProjectScope: nil = semua project (superadmin); selain itu project aktif.
func (a Actor) ProjectScope() *uuid.UUID {
	if a.SuperAdmin {
		return nil
	}
	return a.ProjectID
}

func (a Actor) inProject(p uuid.UUID) bool { return a.ProjectID != nil && *a.ProjectID == p }

// ---------------------------------------------------------------------
// Bantuan "project mana yang dituju permintaan ini" (dipakai banyak fitur)
// ---------------------------------------------------------------------

// TargetProject menentukan project tujuan sebuah permintaan:
//   - bukan superadmin: selalu project aktifnya (requested harus sama atau kosong)
//   - superadmin: requested wajib diisi
func TargetProject(a Actor, requested *uuid.UUID) (uuid.UUID, error) {
	pid := requested
	if !a.SuperAdmin {
		if a.ProjectID == nil || (requested != nil && *requested != *a.ProjectID) {
			return uuid.Nil, apperror.ErrForbidden
		}
		pid = a.ProjectID
	}
	if pid == nil {
		return uuid.Nil, apperror.Invalid("project_id wajib diisi")
	}
	return *pid, nil
}

// Viewable = TargetProject + wajib boleh melihat project itu.
func Viewable(a Actor, requested *uuid.UUID) (uuid.UUID, error) {
	pid, err := TargetProject(a, requested)
	if err != nil {
		return uuid.Nil, err
	}
	if !a.CanView(pid) {
		return uuid.Nil, apperror.ErrForbidden
	}
	return pid, nil
}

// ViewScope: superadmin → filter (boleh nil = semua); lainnya → project aktifnya saja.
func ViewScope(a Actor, filter *uuid.UUID) (*uuid.UUID, error) {
	if a.SuperAdmin {
		return filter, nil
	}
	if a.ProjectID == nil || (filter != nil && *filter != *a.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	return a.ProjectID, nil
}

// ScopeIDs: nil = semua (superadmin); selain itu project aktif saja.
func ScopeIDs(a Actor) []uuid.UUID {
	if a.SuperAdmin {
		return nil
	}
	if a.ProjectID == nil {
		return []uuid.UUID{}
	}
	return []uuid.UUID{*a.ProjectID}
}

// RequireAdminView: halaman System Data (company, project) hanya untuk admin & superadmin.
func RequireAdminView(a Actor) error {
	if a.SuperAdmin || a.Role == RoleAdmin {
		return nil
	}
	return apperror.ErrForbidden
}

// CanDelete: data harian project lain = "tidak ditemukan"; hanya admin & dispatcher yang boleh menghapus.
func CanDelete(a Actor, projectID uuid.UUID) error {
	if !a.CanView(projectID) {
		return apperror.ErrNotFound
	}
	if !a.CanOperate(projectID) {
		return apperror.ErrForbidden
	}
	return nil
}
