// Package service (fitur project) = aturan kelola project: lihat = admin (project aktifnya) & superadmin,
// tambah/ubah = superadmin saja. Juga menyediakan FindActive untuk fitur lain (login, input harian, peta, …).
package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	GetActive(ctx context.Context, id uuid.UUID) (*domain.Project, error)
	List(ctx context.Context, ids []uuid.UUID) ([]domain.ProjectDetail, error)
	GetDetail(ctx context.Context, id uuid.UUID) (*domain.ProjectDetail, error)
	Insert(ctx context.Context, p domain.ProjectDetail, by uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, in domain.ProjectInput, by uuid.UUID) error
}

// CompanyChecker = fitur company (memeriksa company_id ada).
type CompanyChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	repo      Repository
	companies CompanyChecker
}

func New(repo Repository, companies CompanyChecker) *Service {
	return &Service{repo: repo, companies: companies}
}

// FindActive = project aktif berdasarkan id; (nil, nil) bila tidak ada atau nonaktif.
// Tanpa pemeriksaan hak akses — pemanggil (service fitur lain) yang memeriksa.
func (s *Service) FindActive(ctx context.Context, id uuid.UUID) (*domain.Project, error) {
	return s.repo.GetActive(ctx, id)
}

func (s *Service) List(ctx context.Context, a access.Actor) ([]domain.ProjectDetail, error) {
	if err := access.RequireAdminView(a); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, access.ScopeIDs(a))
}

func (s *Service) Save(ctx context.Context, a access.Actor, id *uuid.UUID, in domain.ProjectInput) (*domain.ProjectDetail, error) {
	if !a.SuperAdmin {
		return nil, apperror.ErrForbidden
	}
	var err error
	if in.ProjectCode, err = validate.Code(in.ProjectCode, "project_code"); err != nil {
		return nil, err
	}
	if in.ProjectName, err = validate.Name(in.ProjectName, "project_name", 150); err != nil {
		return nil, err
	}
	if in.Timezone != nil {
		if _, err := time.LoadLocation(strings.TrimSpace(*in.Timezone)); err != nil || strings.TrimSpace(*in.Timezone) == "" {
			return nil, apperror.Invalid("timezone tidak dikenal (contoh: Asia/Jakarta)")
		}
	}
	if in.CompanyID != nil {
		if ok, err := s.companies.Exists(ctx, *in.CompanyID); err != nil {
			return nil, err
		} else if !ok {
			return nil, apperror.Invalid("company tidak ditemukan")
		}
	}
	if id == nil {
		if in.CompanyID == nil || in.ProjectCode == nil || in.ProjectName == nil {
			return nil, apperror.Invalid("company_id, project_code dan project_name wajib diisi")
		}
		p := domain.ProjectDetail{CompanyID: *in.CompanyID, ProjectCode: *in.ProjectCode, ProjectName: *in.ProjectName,
			Timezone: "Asia/Jakarta", IsActive: in.IsActive == nil || *in.IsActive}
		if in.Timezone != nil {
			p.Timezone = strings.TrimSpace(*in.Timezone)
		}
		newID, err := s.repo.Insert(ctx, p, a.UserID)
		if err != nil {
			return nil, err
		}
		id = &newID
	} else {
		cur, err := s.repo.GetDetail(ctx, *id)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, apperror.ErrNotFound
		}
		if err := s.repo.Update(ctx, *id, in, a.UserID); err != nil {
			return nil, err
		}
	}
	return s.repo.GetDetail(ctx, *id)
}
