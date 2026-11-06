// Package service (fitur activation) = status operasi harian equipment (Equipment Activation).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher project, superadmin.
// Equipment dicek lewat fitur equipment (hak lihat ikut aturan equipment).
package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/activation/domain"
	equipmentdomain "simha/core-api/internal/equipment/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f domain.ActivationFilter) ([]domain.Activation, error)
	Get(ctx context.Context, id int64) (*domain.Activation, error)
	Upsert(ctx context.Context, equipmentID uuid.UUID, date, status string, desc *string, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, id int64, date, status *string, desc *string, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.ActivationRow, by uuid.UUID, dryRun bool) (int, int, error)
}

// EquipmentGetter = fitur equipment: equipment yang boleh DILIHAT pemanggil (lainnya ErrNotFound).
type EquipmentGetter interface {
	Get(ctx context.Context, a access.Actor, id uuid.UUID) (*equipmentdomain.Equipment, error)
}

type Service struct {
	repo      Repository
	equipment EquipmentGetter
}

func New(repo Repository, equipment EquipmentGetter) *Service {
	return &Service{repo: repo, equipment: equipment}
}

// ActivationInput = isian aktivasi. Pada ubah, nil = tidak diubah.
type ActivationInput struct {
	EquipmentID    *uuid.UUID `json:"equipment_id"`
	ActivationDate *string    `json:"activation_date"` // YYYY-MM-DD
	Status         *string    `json:"status"`          // active | inactive
	Description    *string    `json:"description"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f domain.ActivationFilter) ([]domain.Activation, error) {
	scope, err := access.ViewScope(a, f.ProjectID)
	if err != nil {
		return nil, err
	}
	f.ProjectID = scope
	for _, d := range []string{f.DateFrom, f.DateTo} {
		if d != "" && !validate.IsDate(d) {
			return nil, apperror.Invalid("tanggal harus format YYYY-MM-DD")
		}
	}
	if f.Status != "" && f.Status != domain.ActivationActive && f.Status != domain.ActivationInactive {
		return nil, apperror.Invalid("status harus active atau inactive")
	}
	return s.repo.List(ctx, f)
}

// Set menyimpan status satu equipment pada satu tanggal. Bila sudah ada → diperbarui
// (sama seperti sistem lama: satu baris per equipment per tanggal). created=true bila baris baru.
func (s *Service) Set(ctx context.Context, a access.Actor, in ActivationInput) (*domain.Activation, bool, error) {
	if in.EquipmentID == nil || in.ActivationDate == nil || in.Status == nil {
		return nil, false, apperror.Invalid("equipment_id, activation_date, dan status wajib diisi")
	}
	eq, err := s.equipment.Get(ctx, a, *in.EquipmentID)
	if err != nil {
		return nil, false, err
	}
	if !a.CanOperate(eq.ProjectID) {
		return nil, false, apperror.ErrForbidden
	}
	if err := checkActivation(in); err != nil {
		return nil, false, err
	}
	id, created, err := s.repo.Upsert(ctx, eq.ID, *in.ActivationDate, *in.Status, validate.TrimPtr(in.Description), a.UserID)
	if err != nil {
		return nil, false, err
	}
	act, err := s.repo.Get(ctx, id)
	return act, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, in ActivationInput) (*domain.Activation, error) {
	cur, err := s.activationFor(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if in.EquipmentID != nil && *in.EquipmentID != cur.EquipmentID {
		return nil, apperror.Invalid("equipment tidak bisa diganti; hapus lalu buat baris baru")
	}
	if err := checkActivation(in); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, id, in.ActivationDate, in.Status, validate.TrimPtr(in.Description), a.UserID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, a access.Actor, id int64) error {
	if _, err := s.activationFor(ctx, a, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// activationFor mengambil aktivasi yang boleh DIUBAH oleh a.
func (s *Service) activationFor(ctx context.Context, a access.Actor, id int64) (*domain.Activation, error) {
	act, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if act == nil || !a.CanView(act.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	if !a.CanOperate(act.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	return act, nil
}

func checkActivation(in ActivationInput) error {
	if in.ActivationDate != nil && !validate.IsDate(*in.ActivationDate) {
		return apperror.Invalid("activation_date harus format YYYY-MM-DD")
	}
	if in.Status != nil && *in.Status != domain.ActivationActive && *in.Status != domain.ActivationInactive {
		return apperror.Invalid("status harus active atau inactive")
	}
	if in.Description != nil && utf8.RuneCountInString(strings.TrimSpace(*in.Description)) > 255 {
		return apperror.Invalid("description maks. 255 karakter")
	}
	return nil
}
