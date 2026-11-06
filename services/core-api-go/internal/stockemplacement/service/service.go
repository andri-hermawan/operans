// Package service (fitur stockemplacement) = stok cargo harian per project (ton), satu baris per partner per tanggal operasional (ADR 0007).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Kolom total disimpan apa adanya (tidak dihitung ulang).
package service

import (
	"context"
	"math"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/shared/validate"
	"simha/core-api/internal/stockemplacement/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.StockEmplacement, error)
	Get(ctx context.Context, id int64) (*domain.StockEmplacement, error)
	Upsert(ctx context.Context, e domain.StockEmplacement, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, e domain.StockEmplacement, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.StockEmplacement, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// StockEmplacementInput = isian form stok cargo (ton). Pada ubah (PATCH), nil = tidak diubah.
type StockEmplacementInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	Partner         *string    `json:"partner"`
	OperationalDate *string    `json:"operational_date"`
	Info            *string    `json:"info"`
	StockInitial    *float64   `json:"stock_cargo_initial"`
	StockScrapper   *float64   `json:"stock_cargo_scrapper"`
	StockCrusher    *float64   `json:"stock_cargo_crusher"`
	Shift1          *float64   `json:"shift_1"`
	Shift2          *float64   `json:"shift_2"`
	HaulingCargo    *float64   `json:"hauling_cargo"`
	TotalInitial    *float64   `json:"total_cargo_initial"`
	TlsOut1         *float64   `json:"tls_out_trainset_1"`
	TlsOut2         *float64   `json:"tls_out_trainset_2"`
	TlsOut3         *float64   `json:"tls_out_trainset_3"`
	TlsOut4         *float64   `json:"tls_out_trainset_4"`
	TlsOut5         *float64   `json:"tls_out_trainset_5"`
	TotalTlsOut     *float64   `json:"total_tls_out"`
	RemainingCargo  *float64   `json:"remaining_cargo"`
}

// amounts = kolom angka input, urutan = domain.AmountColumns.
func (in *StockEmplacementInput) amounts() []*float64 {
	return []*float64{in.StockInitial, in.StockScrapper, in.StockCrusher, in.Shift1, in.Shift2, in.HaulingCargo,
		in.TotalInitial, in.TlsOut1, in.TlsOut2, in.TlsOut3, in.TlsOut4, in.TlsOut5, in.TotalTlsOut, in.RemainingCargo}
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.StockEmplacement, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: project + partner + tanggal sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in StockEmplacementInput) (*domain.StockEmplacement, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	e := domain.StockEmplacement{ProjectID: pd.Project.ID}
	if errs := applyStock(&e, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, e, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch StockEmplacementInput) (*domain.StockEmplacement, error) {
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
	in := StockEmplacementInput{Partner: &cur.Partner, OperationalDate: &cur.OperationalDate, Info: cur.Info,
		StockInitial: cur.StockInitial, StockScrapper: cur.StockScrapper, StockCrusher: cur.StockCrusher,
		Shift1: cur.Shift1, Shift2: cur.Shift2, HaulingCargo: cur.HaulingCargo, TotalInitial: cur.TotalInitial,
		TlsOut1: cur.TlsOut1, TlsOut2: cur.TlsOut2, TlsOut3: cur.TlsOut3, TlsOut4: cur.TlsOut4, TlsOut5: cur.TlsOut5,
		TotalTlsOut: cur.TotalTlsOut, RemainingCargo: cur.RemainingCargo}
	dailyinput.Overlay(&in, patch)
	if errs := applyStock(cur, in); len(errs) > 0 {
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

// applyStock memvalidasi isian LENGKAP ke e: tanggal wajib, angka ton ≥ 0 (2 desimal, boleh kosong).
func applyStock(e *domain.StockEmplacement, in StockEmplacementInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	if v, ok := dailyinput.Required(in.Partner, "partner", &errs); ok {
		if utf8.RuneCountInString(v) > 50 {
			errs = append(errs, dailyinput.RowIssue{Column: "partner", Message: "partner maks. 50 karakter"})
		}
		e.Partner = v
	}
	if v, ok := dailyinput.Required(in.OperationalDate, "operational_date", &errs); ok {
		if d, err := excel.ParseDate(v); err != nil {
			errs = append(errs, dailyinput.RowIssue{Column: "operational_date", Message: err.Error()})
		} else {
			e.OperationalDate = d
		}
	}
	e.Info = dailyinput.Optional(in.Info, "info", 10, &errs)
	for i, p := range e.Amounts() {
		*p = ton(in.amounts()[i], domain.AmountColumns[i], &errs)
	}
	return errs
}

// ton: angka ≥ 0 dibulatkan 2 desimal; kosong → nil.
func ton(v *float64, col string, errs *[]dailyinput.RowIssue) *float64 {
	switch {
	case v == nil || math.IsNaN(*v): // NaN sudah dilaporkan pengurai Excel
		return nil
	case *v < 0 || *v >= 1e12:
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " harus 0 atau lebih"})
		return nil
	}
	r := validate.Round2(*v)
	return &r
}
