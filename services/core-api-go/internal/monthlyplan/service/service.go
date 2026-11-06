// Package service (fitur monthlyplan) = Plan Monthly: target per tanggal dalam satu bulan (ADR 0014).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher (sama dengan input harian). Total disimpan apa adanya;
// kosong → jumlah tanggal. Tanggal yang tidak ada pada bulan itu (mis. 31 Juni) harus 0.
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"simha/core-api/internal/monthlyplan/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/plan"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f plan.Filter) ([]domain.MonthlyPlan, error)
	Get(ctx context.Context, id int64) (*domain.MonthlyPlan, error)
	Upsert(ctx context.Context, p domain.MonthlyPlan, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, p domain.MonthlyPlan, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.MonthlyPlan, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// MonthlyPlanInput = isian form Plan Monthly. Pada ubah (PATCH), nil = tidak diubah.
// Tanggal (day_01..day_31) ikut dibaca dari JSON datar oleh UnmarshalJSON.
type MonthlyPlanInput struct {
	ProjectID   *uuid.UUID `json:"project_id"` // superadmin
	Partner     *string    `json:"partner"`
	PlanYear    *int       `json:"plan_year"`
	PlanMonth   *int       `json:"plan_month"`
	Description *string    `json:"description"`
	PlanName    *string    `json:"plan_name"`
	Uom         *string    `json:"uom"`
	Total       *float64   `json:"total"` // kosong → jumlah tanggal
	Days        []*float64 `json:"-"`     // urutan = domain.DayColumns
}

var knownFields = []string{"project_id", "partner", "plan_year", "plan_month", "description", "plan_name", "uom", "total"}

// UnmarshalJSON membaca isian tetap seperti biasa + 31 kolom tanggal datar; isian yang tidak dikenal ditolak.
func (in *MonthlyPlanInput) UnmarshalJSON(b []byte) error {
	type plain MonthlyPlanInput // tanpa method → tidak memanggil UnmarshalJSON lagi
	if err := json.Unmarshal(b, (*plain)(in)); err != nil {
		return err
	}
	days, err := plan.SplitInput(b, knownFields, domain.DayColumns)
	in.Days = days
	return err
}

func (s *Service) List(ctx context.Context, a access.Actor, f plan.Filter) ([]domain.MonthlyPlan, error) {
	if err := plan.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: mitra + tahun + bulan + deskripsi + nama sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in MonthlyPlanInput) (*domain.MonthlyPlan, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	p := domain.MonthlyPlan{ProjectID: pd.Project.ID}
	if errs := applyMonthly(&p, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, p, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch MonthlyPlanInput) (*domain.MonthlyPlan, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || !a.CanView(cur.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	if _, err := s.daily.Load(ctx, a, &cur.ProjectID, true); err != nil {
		return nil, err
	}
	in := MonthlyPlanInput{Partner: &cur.Partner, PlanYear: &cur.PlanYear, PlanMonth: &cur.PlanMonth,
		Description: &cur.Description, PlanName: &cur.PlanName, Uom: &cur.Uom, Total: &cur.Total, Days: plan.Pointers(cur.Days[:])}
	dailyinput.Overlay(&in, patch)
	days, daysChanged := plan.MergeSeries(in.Days, patch.Days)
	in.Days = days
	if patch.Total == nil && daysChanged {
		in.Total = nil // tanggal diubah tanpa total baru → total dihitung ulang dari jumlah tanggal
	}
	if errs := applyMonthly(cur, in); len(errs) > 0 {
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

// applyMonthly memvalidasi isian LENGKAP ke p: mitra, tahun, bulan, deskripsi, nama, uom wajib; angka ≥ 0 (2 desimal);
// tanggal di luar panjang bulan harus 0.
func applyMonthly(p *domain.MonthlyPlan, in MonthlyPlanInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	p.Partner = plan.Text(in.Partner, "partner", 50, &errs)
	p.PlanYear = plan.Year(in.PlanYear, &errs)
	p.PlanMonth = plan.Month(in.PlanMonth, &errs)
	p.Description = plan.Text(in.Description, "description", 100, &errs)
	p.PlanName = plan.Text(in.PlanName, "plan_name", 100, &errs)
	p.Uom = plan.Text(in.Uom, "uom", 20, &errs)
	days := plan.Amounts(in.Days, domain.DayColumns, &errs)
	copy(p.Days[:], days)
	if p.PlanYear != 0 && p.PlanMonth != 0 {
		for d := domain.DaysInMonth(p.PlanYear, p.PlanMonth); d < len(days); d++ {
			if days[d] != 0 {
				errs = append(errs, dailyinput.RowIssue{Column: domain.DayColumns[d],
					Message: fmt.Sprintf("%s harus 0: bulan %d/%d hanya punya %d hari", domain.DayColumns[d], p.PlanMonth, p.PlanYear,
						domain.DaysInMonth(p.PlanYear, p.PlanMonth))})
			}
		}
	}
	p.Total = plan.Total(in.Total, plan.Sum(days), &errs)
	return errs
}
