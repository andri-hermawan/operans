// Package service (fitur workhour) = WH & Notification: kejadian jam kerja unit (ADR 0015).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Jam mulai & selesai wajib; total jam kosong → dihitung.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/workhour/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.WorkHourNotification, error)
	Get(ctx context.Context, id int64) (*domain.WorkHourNotification, error)
	Upsert(ctx context.Context, t domain.WorkHourNotification, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, t domain.WorkHourNotification, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.WorkHourNotification, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// WorkHourInput = isian form WH & Notification. Jam = HH:MM. Pada ubah (PATCH), nil = tidak diubah.
type WorkHourInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"`
	EquipmentCode   *string    `json:"equipment_code"`
	Category        *string    `json:"category"`
	SubCategory     *string    `json:"sub_category"`
	TimeStart       *string    `json:"time_start"`
	TimeEnd         *string    `json:"time_end"`
	TotalHours      *float64   `json:"total_hours"` // kosong → time_end − time_start
	Dedicated       *string    `json:"dedicated"`
	Remark          *string    `json:"remark"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.WorkHourNotification, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: unit + kategori + jam mulai sama pada tanggal & shift yang sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in WorkHourInput) (*domain.WorkHourNotification, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	t := domain.WorkHourNotification{ProjectID: pd.Project.ID}
	if errs := applyWorkHour(pd, &t, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, t, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch WorkHourInput) (*domain.WorkHourNotification, error) {
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
	in := WorkHourInput{OperationalDate: &cur.OperationalDate, Shift: &cur.ShiftCode, EquipmentCode: &cur.EquipmentCode,
		Category: &cur.Category, SubCategory: cur.SubCategory, TimeStart: dailyinput.ClockOf(&cur.StartedAt, pd.Loc),
		TimeEnd: dailyinput.ClockOf(cur.EndedAt, pd.Loc), TotalHours: cur.TotalHours, Dedicated: cur.Dedicated, Remark: cur.Remark}
	if (patch.TimeStart != nil || patch.TimeEnd != nil) && patch.TotalHours == nil {
		in.TotalHours = nil // jam berubah tanpa total baru → total dihitung ulang
	}
	dailyinput.Overlay(&in, patch)
	if errs := applyWorkHour(pd, cur, in); len(errs) > 0 {
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

func blank(p *string) bool { return p == nil || strings.TrimSpace(*p) == "" }

// applyWorkHour memvalidasi isian LENGKAP ke t: tanggal, shift, unit, kategori, jam mulai & selesai wajib.
func applyWorkHour(pd *dailyinput.ProjectData, t *domain.WorkHourNotification, in WorkHourInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, dateOK := pd.ApplyDateShift(&t.OperationalDate, in.OperationalDate, in.Shift, &errs)
	t.ShiftID, t.ShiftCode, t.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&t.EquipmentCode, &t.EquipmentID, v, &errs, "equipment_code")
	}
	if v, ok := dailyinput.Required(in.Category, "category", &errs); ok {
		if len([]rune(v)) > 100 {
			errs = append(errs, dailyinput.RowIssue{Column: "category", Message: "category maks. 100 karakter"})
		}
		t.Category = v
	}
	t.SubCategory = dailyinput.Optional(in.SubCategory, "sub_category", 100, &errs)
	t.Dedicated = dailyinput.Optional(in.Dedicated, "dedicated", 100, &errs)
	t.Remark = dailyinput.Optional(in.Remark, "remark", 255, &errs)

	if blank(in.TimeStart) {
		errs = append(errs, dailyinput.RowIssue{Column: "time_start", Message: "time_start wajib diisi"})
	}
	if blank(in.TimeEnd) {
		errs = append(errs, dailyinput.RowIssue{Column: "time_end", Message: "time_end wajib diisi"})
	}
	start, end, hours := pd.Range(sh, t.OperationalDate, dateOK, in.TimeStart, in.TimeEnd, in.TotalHours,
		dailyinput.RangeCols{Start: "time_start", End: "time_end", Hours: "total_hours"}, &errs)
	if start != nil {
		t.StartedAt = *start
	}
	t.EndedAt, t.TotalHours = end, hours
	return errs
}
