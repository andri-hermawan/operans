// Package service (fitur activation) = status operasi harian unit (Equipment Activation).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher project, superadmin.
// Seperti tiket timbangan, unit cukup berupa kode: tidak wajib ada di master equipment (equipment_id terisi bila ada).
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/activation/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f domain.ActivationFilter) ([]domain.Activation, error)
	Get(ctx context.Context, id int64) (*domain.Activation, error)
	Upsert(ctx context.Context, a domain.ActivationRow, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, id int64, date, status *string, desc *string, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.ActivationRow, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// ActivationInput = isian aktivasi. Pada ubah, nil = tidak diubah.
type ActivationInput struct {
	ProjectID      *uuid.UUID `json:"project_id"`     // superadmin
	EquipmentCode  *string    `json:"equipment_code"` // kode unit (huruf besar), boleh belum ada di master
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

// Set menyimpan status satu unit pada satu tanggal. Bila sudah ada → diperbarui
// (satu baris per unit per tanggal). created=true bila baris baru.
func (s *Service) Set(ctx context.Context, a access.Actor, in ActivationInput) (*domain.Activation, bool, error) {
	if in.EquipmentCode == nil || in.ActivationDate == nil || in.Status == nil {
		return nil, false, apperror.Invalid("equipment_code, activation_date, dan status wajib diisi")
	}
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	row := domain.ActivationRow{ProjectID: pd.Project.ID}
	if errs := applyActivation(pd, &row, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, row, a.UserID)
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
	if in.EquipmentCode != nil && strings.ToUpper(strings.TrimSpace(*in.EquipmentCode)) != cur.EquipmentCode {
		return nil, apperror.Invalid("unit tidak bisa diganti; hapus lalu buat baris baru")
	}
	in.EquipmentCode = &cur.EquipmentCode
	pd, err := s.daily.Load(ctx, a, &cur.ProjectID, true)
	if err != nil {
		return nil, err
	}
	var row domain.ActivationRow
	if errs := applyActivation(pd, &row, in); len(errs) > 0 {
		return nil, apperror.Invalid("%s", errs[0].Message)
	}
	if err := s.repo.Update(ctx, id, in.ActivationDate, in.Status, row.Description, a.UserID); err != nil {
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
	if act == nil {
		return nil, apperror.ErrNotFound
	}
	if err := access.CanDelete(a, act.ProjectID); err != nil {
		return nil, err
	}
	return act, nil
}

// applyActivation memvalidasi isian ke row. Unit: kode huruf besar (maks. 50), boleh belum ada di master.
// Pada ubah (PATCH) tanggal/status yang nil tidak diperiksa dan dibiarkan kosong di row (repository memakai COALESCE).
func applyActivation(pd *dailyinput.ProjectData, row *domain.ActivationRow, in ActivationInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&row.EquipmentCode, &row.EquipmentID, v, &errs, "equipment_code")
	}
	if in.ActivationDate != nil {
		if !validate.IsDate(*in.ActivationDate) {
			errs = append(errs, dailyinput.RowIssue{Column: "activation_date", Message: "activation_date harus format YYYY-MM-DD"})
		}
		row.Date = *in.ActivationDate
	}
	if in.Status != nil {
		if *in.Status != domain.ActivationActive && *in.Status != domain.ActivationInactive {
			errs = append(errs, dailyinput.RowIssue{Column: "status", Message: "status harus active atau inactive"})
		}
		row.Status = *in.Status
	}
	row.Description = dailyinput.Optional(in.Description, "description", 255, &errs)
	return errs
}
