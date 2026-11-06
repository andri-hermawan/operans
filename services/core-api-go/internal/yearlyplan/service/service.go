// Package service (fitur yearlyplan) = Plan Yearly: target per bulan dalam satu tahun (ADR 0014).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher (sama dengan input harian). Total disimpan apa adanya;
// kosong → jumlah bulan.
package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/plan"
	"simha/core-api/internal/yearlyplan/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f plan.Filter) ([]domain.YearlyPlan, error)
	Get(ctx context.Context, id int64) (*domain.YearlyPlan, error)
	Upsert(ctx context.Context, p domain.YearlyPlan, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, p domain.YearlyPlan, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.YearlyPlan, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// YearlyPlanInput = isian form Plan Yearly. Pada ubah (PATCH), nil = tidak diubah.
// Bulan (jan..dec) ikut dibaca dari JSON datar oleh UnmarshalJSON.
type YearlyPlanInput struct {
	ProjectID   *uuid.UUID `json:"project_id"` // superadmin
	Partner     *string    `json:"partner"`
	PlanYear    *int       `json:"plan_year"`
	Location    *string    `json:"location"`
	Description *string    `json:"description"`
	PlanName    *string    `json:"plan_name"`
	Total       *float64   `json:"total"` // kosong → jumlah bulan
	Months      []*float64 `json:"-"`     // urutan = domain.MonthColumns
}

var knownFields = []string{"project_id", "partner", "plan_year", "location", "description", "plan_name", "total"}

// UnmarshalJSON membaca isian tetap seperti biasa + 12 kolom bulan datar; isian yang tidak dikenal ditolak.
func (in *YearlyPlanInput) UnmarshalJSON(b []byte) error {
	type plain YearlyPlanInput // tanpa method → tidak memanggil UnmarshalJSON lagi
	if err := json.Unmarshal(b, (*plain)(in)); err != nil {
		return err
	}
	months, err := plan.SplitInput(b, knownFields, domain.MonthColumns)
	in.Months = months
	return err
}

func (s *Service) List(ctx context.Context, a access.Actor, f plan.Filter) ([]domain.YearlyPlan, error) {
	if err := plan.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: mitra + tahun + lokasi + deskripsi + nama sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in YearlyPlanInput) (*domain.YearlyPlan, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	p := domain.YearlyPlan{ProjectID: pd.Project.ID}
	if errs := applyYearly(&p, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, p, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch YearlyPlanInput) (*domain.YearlyPlan, error) {
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
	in := YearlyPlanInput{Partner: &cur.Partner, PlanYear: &cur.PlanYear, Location: &cur.Location,
		Description: &cur.Description, PlanName: &cur.PlanName, Total: &cur.Total, Months: plan.Pointers(cur.Months[:])}
	dailyinput.Overlay(&in, patch)
	months, monthsChanged := plan.MergeSeries(in.Months, patch.Months)
	in.Months = months
	if patch.Total == nil && monthsChanged {
		in.Total = nil // bulan diubah tanpa total baru → total dihitung ulang dari jumlah bulan
	}
	if errs := applyYearly(cur, in); len(errs) > 0 {
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

// applyYearly memvalidasi isian LENGKAP ke p: mitra, tahun, lokasi, deskripsi, nama wajib; angka ≥ 0 (2 desimal).
func applyYearly(p *domain.YearlyPlan, in YearlyPlanInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	p.Partner = plan.Text(in.Partner, "partner", 50, &errs)
	p.PlanYear = plan.Year(in.PlanYear, &errs)
	p.Location = plan.Text(in.Location, "location", 100, &errs)
	p.Description = plan.Text(in.Description, "description", 100, &errs)
	p.PlanName = plan.Text(in.PlanName, "plan_name", 100, &errs)
	months := plan.Amounts(in.Months, domain.MonthColumns, &errs)
	copy(p.Months[:], months)
	p.Total = plan.Total(in.Total, plan.Sum(months), &errs)
	return errs
}
