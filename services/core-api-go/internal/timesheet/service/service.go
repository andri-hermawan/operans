// Package service (fitur timesheet) = Time Sheet: HM awal/akhir unit per tanggal + shift (ADR 0015).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Project, shift & master equipment dimuat lewat
// dailyinput.Loader. HM total disimpan apa adanya; kosong → HM akhir − HM awal.
package service

import (
	"context"
	"math"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
	"simha/core-api/internal/timesheet/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.TimeSheet, error)
	Get(ctx context.Context, id int64) (*domain.TimeSheet, error)
	Upsert(ctx context.Context, t domain.TimeSheet, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, t domain.TimeSheet, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.TimeSheet, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// TimeSheetInput = isian form Time Sheet. HM dalam jam. Pada ubah (PATCH), nil = tidak diubah.
type TimeSheetInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"`
	EquipmentCode   *string    `json:"equipment_code"`
	OperatorNik     *string    `json:"operator_nik"`
	OperatorName    *string    `json:"operator_name"`
	HmStart         *float64   `json:"hm_start"`
	HmEnd           *float64   `json:"hm_end"`
	HmTotal         *float64   `json:"hm_total"` // kosong → HM akhir − HM awal
	Remark          *string    `json:"remark"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.TimeSheet, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + shift + unit sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in TimeSheetInput) (*domain.TimeSheet, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	t := domain.TimeSheet{ProjectID: pd.Project.ID}
	if errs := applyTimeSheet(pd, &t, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, t, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch TimeSheetInput) (*domain.TimeSheet, error) {
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
	in := TimeSheetInput{OperationalDate: &cur.OperationalDate, Shift: &cur.ShiftCode, EquipmentCode: &cur.EquipmentCode,
		OperatorNik: cur.OperatorNik, OperatorName: cur.OperatorName, HmStart: cur.HmStart, HmEnd: cur.HmEnd, HmTotal: cur.HmTotal,
		Remark: cur.Remark}
	if (patch.HmStart != nil || patch.HmEnd != nil) && patch.HmTotal == nil {
		in.HmTotal = nil // HM berubah tanpa total baru → total dihitung ulang
	}
	dailyinput.Overlay(&in, patch)
	if errs := applyTimeSheet(pd, cur, in); len(errs) > 0 {
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

// applyTimeSheet memvalidasi isian LENGKAP ke t: tanggal, shift, unit wajib; HM ≥ 0 (2 desimal); HM akhir ≥ HM awal.
func applyTimeSheet(pd *dailyinput.ProjectData, t *domain.TimeSheet, in TimeSheetInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, _ := pd.ApplyDateShift(&t.OperationalDate, in.OperationalDate, in.Shift, &errs)
	t.ShiftID, t.ShiftCode, t.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&t.EquipmentCode, &t.EquipmentID, v, &errs, "equipment_code")
	}
	t.OperatorNik = dailyinput.Optional(in.OperatorNik, "operator_nik", 30, &errs)
	t.OperatorName = dailyinput.Optional(in.OperatorName, "operator_name", 100, &errs)
	t.Remark = dailyinput.Optional(in.Remark, "remark", 255, &errs)

	t.HmStart = hourMeter(in.HmStart, "hm_start", &errs)
	t.HmEnd = hourMeter(in.HmEnd, "hm_end", &errs)
	t.HmTotal = hourMeter(in.HmTotal, "hm_total", &errs)
	if t.HmStart != nil && t.HmEnd != nil {
		if *t.HmEnd < *t.HmStart {
			errs = append(errs, dailyinput.RowIssue{Column: "hm_end", Message: "hm_end lebih kecil dari hm_start"})
		} else if t.HmTotal == nil {
			total := validate.Round2(*t.HmEnd - *t.HmStart)
			t.HmTotal = &total
		}
	}
	return errs
}

// hourMeter: angka HM ≥ 0 dibulatkan 2 desimal; kosong → nil.
func hourMeter(v *float64, col string, errs *[]dailyinput.RowIssue) *float64 {
	switch {
	case v == nil || math.IsNaN(*v): // NaN sudah dilaporkan pengurai Excel
		return nil
	case *v < 0 || *v >= 1e10:
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " harus 0 atau lebih"})
		return nil
	}
	r := validate.Round2(*v)
	return &r
}
