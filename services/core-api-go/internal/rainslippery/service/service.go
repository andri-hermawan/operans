// Package service (fitur rainslippery) = Rain & Slippery: hujan, tunggu licin, dan licin per lokasi + shift (ADR 0015).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Tiga pasangan jam; semuanya boleh kosong.
package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/rainslippery/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.RainSlipperyLog, error)
	Get(ctx context.Context, id int64) (*domain.RainSlipperyLog, error)
	Upsert(ctx context.Context, t domain.RainSlipperyLog, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, t domain.RainSlipperyLog, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.RainSlipperyLog, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// RainSlipperyInput = isian form Rain & Slippery. Jam = HH:MM. Pada ubah (PATCH), nil = tidak diubah.
type RainSlipperyInput struct {
	ProjectID         *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate   *string    `json:"operational_date"`
	Shift             *string    `json:"shift"`
	Location          *string    `json:"location"`
	RainStart         *string    `json:"rain_start"`
	RainStop          *string    `json:"rain_stop"`
	RainHours         *float64   `json:"rain_hours"`
	WaitSlipperyStart *string    `json:"wait_slippery_start"`
	WaitSlipperyStop  *string    `json:"wait_slippery_stop"`
	WaitSlipperyHours *float64   `json:"wait_slippery_hours"`
	SlipperyStart     *string    `json:"slippery_start"`
	SlipperyStop      *string    `json:"slippery_stop"`
	SlipperyHours     *float64   `json:"slippery_hours"`
	RainfallMm        *float64   `json:"rainfall_mm"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.RainSlipperyLog, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + shift + lokasi sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in RainSlipperyInput) (*domain.RainSlipperyLog, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	t := domain.RainSlipperyLog{ProjectID: pd.Project.ID}
	if errs := applyRainSlippery(pd, &t, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, t, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch RainSlipperyInput) (*domain.RainSlipperyLog, error) {
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
	in := rainSlipperyToInput(cur, pd)
	// Jam salah satu pasangan berubah tanpa total baru → total pasangan itu dihitung ulang.
	if (patch.RainStart != nil || patch.RainStop != nil) && patch.RainHours == nil {
		in.RainHours = nil
	}
	if (patch.WaitSlipperyStart != nil || patch.WaitSlipperyStop != nil) && patch.WaitSlipperyHours == nil {
		in.WaitSlipperyHours = nil
	}
	if (patch.SlipperyStart != nil || patch.SlipperyStop != nil) && patch.SlipperyHours == nil {
		in.SlipperyHours = nil
	}
	dailyinput.Overlay(&in, patch)
	if errs := applyRainSlippery(pd, cur, in); len(errs) > 0 {
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

// rainSlipperyToInput mengubah data tersimpan kembali menjadi isian (untuk digabung dengan PATCH).
func rainSlipperyToInput(t *domain.RainSlipperyLog, pd *dailyinput.ProjectData) RainSlipperyInput {
	return RainSlipperyInput{OperationalDate: &t.OperationalDate, Shift: &t.ShiftCode, Location: t.Location,
		RainStart: dailyinput.ClockOf(t.RainStartAt, pd.Loc), RainStop: dailyinput.ClockOf(t.RainStopAt, pd.Loc), RainHours: t.RainHours,
		WaitSlipperyStart: dailyinput.ClockOf(t.WaitSlipperyStartAt, pd.Loc), WaitSlipperyStop: dailyinput.ClockOf(t.WaitSlipperyStopAt, pd.Loc),
		WaitSlipperyHours: t.WaitSlipperyHours,
		SlipperyStart:     dailyinput.ClockOf(t.SlipperyStartAt, pd.Loc), SlipperyStop: dailyinput.ClockOf(t.SlipperyStopAt, pd.Loc),
		SlipperyHours: t.SlipperyHours, RainfallMm: t.RainfallMm}
}

// applyRainSlippery memvalidasi isian LENGKAP ke t: tanggal & shift wajib; tiga pasangan jam opsional.
func applyRainSlippery(pd *dailyinput.ProjectData, t *domain.RainSlipperyLog, in RainSlipperyInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, dateOK := pd.ApplyDateShift(&t.OperationalDate, in.OperationalDate, in.Shift, &errs)
	t.ShiftID, t.ShiftCode, t.ShiftName = sh.ID, sh.Code, sh.Name
	t.Location = dailyinput.Optional(in.Location, "location", 100, &errs)
	t.RainfallMm = dailyinput.Amount(in.RainfallMm, "rainfall_mm", 2, &errs)

	d := t.OperationalDate
	t.RainStartAt, t.RainStopAt, t.RainHours = pd.Range(sh, d, dateOK, in.RainStart, in.RainStop, in.RainHours,
		dailyinput.RangeCols{Start: "rain_start", End: "rain_stop", Hours: "rain_hours"}, &errs)
	t.WaitSlipperyStartAt, t.WaitSlipperyStopAt, t.WaitSlipperyHours = pd.Range(sh, d, dateOK, in.WaitSlipperyStart, in.WaitSlipperyStop,
		in.WaitSlipperyHours, dailyinput.RangeCols{Start: "wait_slippery_start", End: "wait_slippery_stop", Hours: "wait_slippery_hours"}, &errs)
	t.SlipperyStartAt, t.SlipperyStopAt, t.SlipperyHours = pd.Range(sh, d, dateOK, in.SlipperyStart, in.SlipperyStop,
		in.SlipperyHours, dailyinput.RangeCols{Start: "slippery_start", End: "slippery_stop", Hours: "slippery_hours"}, &errs)
	return errs
}
