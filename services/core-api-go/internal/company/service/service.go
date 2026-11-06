// Package service (fitur company) = aturan kelola company: lihat = admin (company project aktifnya) &
// superadmin, tambah/ubah = superadmin saja.
package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/company/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectIDs []uuid.UUID) ([]domain.Company, error)
	Insert(ctx context.Context, code, name string, active bool, by uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, in domain.CompanyInput, by uuid.UUID) (bool, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, a access.Actor) ([]domain.Company, error) {
	if err := access.RequireAdminView(a); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, access.ScopeIDs(a))
}

func (s *Service) Save(ctx context.Context, a access.Actor, id *uuid.UUID, in domain.CompanyInput) error {
	if !a.SuperAdmin {
		return apperror.ErrForbidden
	}
	var err error
	if in.CompanyCode, err = validate.Code(in.CompanyCode, "company_code"); err != nil {
		return err
	}
	if in.CompanyName, err = validate.Name(in.CompanyName, "company_name", 150); err != nil {
		return err
	}
	if id == nil {
		if in.CompanyCode == nil || in.CompanyName == nil {
			return apperror.Invalid("company_code dan company_name wajib diisi")
		}
		_, err := s.repo.Insert(ctx, *in.CompanyCode, *in.CompanyName, in.IsActive == nil || *in.IsActive, a.UserID)
		return err
	}
	ok, err := s.repo.Update(ctx, *id, in, a.UserID)
	if err == nil && !ok {
		return apperror.ErrNotFound
	}
	return err
}

// Exists dipakai fitur project (memeriksa company_id saat menyimpan project).
func (s *Service) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.Exists(ctx, id)
}
