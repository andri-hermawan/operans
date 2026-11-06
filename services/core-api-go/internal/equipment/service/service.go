// Package service (fitur equipment) = aturan master unit: lihat = semua role di project,
// tambah/ubah = admin project & superadmin. Juga menyediakan Get & CodeMap untuk fitur lain
// (aktivasi, kalibrasi fuel, input harian).
package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/equipment/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectID *uuid.UUID, q string, includeInactive bool) ([]domain.Equipment, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Equipment, error)
	Insert(ctx context.Context, e domain.Equipment, by uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, in domain.EquipmentInput, by uuid.UUID) error
	CodeMap(ctx context.Context, projectID uuid.UUID) (map[string]uuid.UUID, error)
}

// ProjectFinder = fitur project.
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

type Service struct {
	repo     Repository
	projects ProjectFinder
}

func New(repo Repository, projects ProjectFinder) *Service {
	return &Service{repo: repo, projects: projects}
}

// CodeMap = kode → id semua equipment project (untuk menautkan kode unit di data harian).
func (s *Service) CodeMap(ctx context.Context, projectID uuid.UUID) (map[string]uuid.UUID, error) {
	return s.repo.CodeMap(ctx, projectID)
}

// ---------------------------------------------------------------------
// Equipment — lihat: semua role di project; tambah/ubah: admin project & superadmin
// ---------------------------------------------------------------------

func (s *Service) List(ctx context.Context, a access.Actor, projectFilter *uuid.UUID, q string, includeInactive bool) ([]domain.Equipment, error) {
	scope, err := access.ViewScope(a, projectFilter)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, scope, strings.TrimSpace(q), includeInactive)
}

func (s *Service) Get(ctx context.Context, a access.Actor, id uuid.UUID) (*domain.Equipment, error) {
	e, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil || !a.CanView(e.ProjectID) {
		return nil, apperror.ErrNotFound // equipment project lain = "tidak ditemukan"
	}
	return e, nil
}

func (s *Service) Create(ctx context.Context, a access.Actor, in domain.EquipmentInput) (*domain.Equipment, error) {
	if !a.SuperAdmin {
		in.ProjectID = a.ProjectID // admin selalu di project aktifnya
	}
	if in.ProjectID == nil {
		return nil, apperror.Invalid("project_id wajib diisi")
	}
	if !a.CanManageProject(*in.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	if p, err := s.projects.FindActive(ctx, *in.ProjectID); err != nil {
		return nil, err
	} else if p == nil {
		return nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
	}
	if in.EquipmentCode == nil {
		return nil, apperror.Invalid("equipment_code wajib diisi")
	}
	e := domain.Equipment{ProjectID: *in.ProjectID}
	if err := applyEquipment(&e, in); err != nil {
		return nil, err
	}
	id, err := s.repo.Insert(ctx, e, a.UserID)
	if err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, a access.Actor, id uuid.UUID, in domain.EquipmentInput) (*domain.Equipment, error) {
	cur, err := s.Get(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !a.CanManageProject(cur.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	if in.ProjectID != nil && *in.ProjectID != cur.ProjectID {
		return nil, apperror.Invalid("equipment tidak bisa dipindah project; buat equipment baru di project tujuan")
	}
	tmp := *cur
	if err := applyEquipment(&tmp, in); err != nil {
		return nil, err
	}
	in.EquipmentCode = &tmp.EquipmentCode // versi yang sudah dirapikan
	if err := s.repo.Update(ctx, id, in, a.UserID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}

// applyEquipment memvalidasi & merapikan isian ke e (kode huruf besar, teks dipangkas).
func applyEquipment(e *domain.Equipment, in domain.EquipmentInput) error {
	if in.EquipmentCode != nil {
		code := strings.ToUpper(strings.TrimSpace(*in.EquipmentCode))
		if code == "" || utf8.RuneCountInString(code) > 50 {
			return apperror.Invalid("equipment_code wajib diisi (maks. 50 karakter)")
		}
		e.EquipmentCode = code
	}
	for _, f := range []struct {
		src *string
		dst **string
		max int
		nm  string
	}{
		{in.EquipmentAlias, &e.EquipmentAlias, 150, "equipment_alias"},
		{in.EquipmentType, &e.EquipmentType, 50, "equipment_type"},
		{in.Brand, &e.Brand, 50, "brand"},
		{in.Model, &e.Model, 50, "model"},
		{in.Class, &e.Class, 50, "class"},
	} {
		if f.src == nil {
			continue
		}
		v := strings.TrimSpace(*f.src)
		if utf8.RuneCountInString(v) > f.max {
			return apperror.Invalid("%s maks. %d karakter", f.nm, f.max)
		}
		*f.dst = &v
	}
	if in.IsActive != nil {
		e.IsActive = *in.IsActive
	} else if e.ID == uuid.Nil {
		e.IsActive = true
	}
	return nil
}
