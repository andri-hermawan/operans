// Package service (fitur alerttype) = Alert Category: jenis alert & nilai bawaan ambangnya (ADR 0008).
// Lihat = semua user yang login; ubah = superadmin saja. Find dipakai fitur alertrule.
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/alerttype/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context) ([]domain.AlertType, error)
	Update(ctx context.Context, t domain.AlertType, by uuid.UUID) error
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]domain.AlertType, error) {
	return s.repo.List(ctx)
}

// AlertTypeInput = perubahan jenis alert; nil = tidak diubah.
type AlertTypeInput struct {
	Name                   *string  `json:"name"`
	Description            *string  `json:"description"`
	DefaultDurationMinutes *float64 `json:"default_duration_minutes"`
	DefaultSpeedLimitKph   *float64 `json:"default_speed_limit_kph"`
	DefaultFuelDropLiters  *float64 `json:"default_fuel_drop_liters"`
}

func (s *Service) Update(ctx context.Context, a access.Actor, code string, in AlertTypeInput) (*domain.AlertType, error) {
	if !a.SuperAdmin {
		return nil, apperror.ErrForbidden
	}
	t, err := s.Find(ctx, code)
	if err != nil {
		return nil, err
	}
	if v := validate.TrimPtr(in.Name); v != nil {
		if *v == "" || len([]rune(*v)) > 100 {
			return nil, apperror.Invalid("name wajib diisi (maks. 100 karakter)")
		}
		t.Name = *v
	}
	if v := validate.TrimPtr(in.Description); v != nil {
		if *v == "" || len([]rune(*v)) > 255 {
			return nil, apperror.Invalid("description wajib diisi (maks. 255 karakter)")
		}
		t.Description = *v
	}
	if err := domain.ApplyThresholds(t, &t.DefaultDurationMinutes, &t.DefaultSpeedLimitKph, &t.DefaultFuelDropLiters,
		in.DefaultDurationMinutes, in.DefaultSpeedLimitKph, in.DefaultFuelDropLiters, "default_"); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, *t, a.UserID); err != nil {
		return nil, err
	}
	return s.Find(ctx, code)
}

// Find = satu jenis alert berdasarkan kode (tidak peka huruf besar); ErrNotFound bila tidak dikenal.
func (s *Service) Find(ctx context.Context, code string) (*domain.AlertType, error) {
	types, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		if t.Code == strings.ToLower(strings.TrimSpace(code)) {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("%w: jenis alert %q tidak dikenal", apperror.ErrNotFound, code)
}
