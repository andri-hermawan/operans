// Package service (fitur operator) = Setting Operator: operator yang menjalankan unit per tanggal + shift.
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Project, shift, master equipment &
// daftar pilihan dimuat lewat dailyinput.Loader (service fitur project, shift, equipment, reference).
package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/operator/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
	shiftdomain "simha/core-api/internal/shift/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.OperatorAssignment, error)
	Get(ctx context.Context, id int64) (*domain.OperatorAssignment, error)
	Upsert(ctx context.Context, o domain.OperatorAssignment, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, o domain.OperatorAssignment, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.OperatorAssignment, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// OperatorInput = isian form operator harian. Pada ubah, nil = tidak diubah.
type OperatorInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"` // "Shift 1" / "SHIFT_1" / "1"
	EquipmentCode   *string    `json:"equipment_code"`
	OperatorName    *string    `json:"operator_name"`
	Description     *string    `json:"description"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.OperatorAssignment, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + shift + unit yang sama → diperbarui (seperti sistem lama).
func (s *Service) Set(ctx context.Context, a access.Actor, in OperatorInput) (*domain.OperatorAssignment, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	o := domain.OperatorAssignment{ProjectID: pd.Project.ID}
	if errs := applyOperator(pd, &o, in, true); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, o, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, in OperatorInput) (*domain.OperatorAssignment, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || !a.CanView(cur.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	pd, err := s.daily.Load(ctx, a, &cur.ProjectID, true)
	if err != nil {
		return nil, err
	}
	if errs := applyOperator(pd, cur, in, false); len(errs) > 0 {
		return nil, apperror.Invalid("%s", errs[0].Message)
	}
	if err := s.repo.Update(ctx, *cur, a.UserID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, a access.Actor, id int64) error {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if cur == nil {
		return apperror.ErrNotFound
	}
	if err := access.CanDelete(a, cur.ProjectID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// applyOperator memvalidasi isian ke o. full=true → semua kolom wajib (tambah); false → hanya yang diisi (ubah).
func applyOperator(pd *dailyinput.ProjectData, o *domain.OperatorAssignment, in OperatorInput, full bool) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	need := func(p *string, col string) bool {
		if p == nil || strings.TrimSpace(*p) == "" {
			if full || p != nil {
				errs = append(errs, dailyinput.RowIssue{Column: col, Message: col + " wajib diisi"})
			}
			return false
		}
		return true
	}
	if need(in.OperationalDate, "operational_date") {
		if !validate.IsDate(*in.OperationalDate) {
			errs = append(errs, dailyinput.RowIssue{Column: "operational_date", Message: "operational_date harus format YYYY-MM-DD"})
		} else {
			o.OperationalDate = *in.OperationalDate
		}
	}
	if need(in.Shift, "shift") {
		if sh, ok := shiftdomain.ResolveShift(pd.Shifts, *in.Shift); ok {
			o.ShiftID, o.ShiftCode, o.ShiftName = sh.ID, sh.Code, sh.Name
		} else {
			errs = append(errs, dailyinput.RowIssue{Column: "shift", Message: "shift \"" + *in.Shift + "\" tidak dikenal di project ini"})
		}
	}
	if need(in.EquipmentCode, "equipment_code") {
		pd.SetEquipment(&o.EquipmentCode, &o.EquipmentID, *in.EquipmentCode, &errs, "equipment_code")
	}
	if need(in.OperatorName, "operator_name") {
		o.OperatorName = strings.TrimSpace(*in.OperatorName)
		if utf8.RuneCountInString(o.OperatorName) > 100 {
			errs = append(errs, dailyinput.RowIssue{Column: "operator_name", Message: "operator_name maks. 100 karakter"})
		}
	}
	if in.Description != nil {
		d := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(d) > 255 {
			errs = append(errs, dailyinput.RowIssue{Column: "description", Message: "description maks. 255 karakter"})
		}
		o.Description = &d
	}
	return errs
}
