// Package service (fitur report) = rekap laporan harian Ritase/Payload/Produksi/Breakdown/PA
// (matriks Asset ID × tanggal). Semua role project boleh melihat.
package service

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/report/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	WeighbridgeDaily(ctx context.Context, f domain.ReportFilter) ([]domain.DailyCell, error)
	BreakdownDaily(ctx context.Context, f domain.ReportFilter) ([]domain.DailyCell, error)
	ExecutiveHauling(ctx context.Context, projectID uuid.UUID, date string) ([]domain.ExecutiveRow, error)
	ExecutiveEmplacement(ctx context.Context, projectID uuid.UUID, date string) ([]domain.ExecutiveEmplacementRow, error)
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

// ReportQuery = filter laporan dari HTTP.
type ReportQuery struct {
	ProjectID *uuid.UUID
	From, To  string
	ShiftID   *uuid.UUID
	Codes     []string
}

var ymd = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (s *Service) filter(a access.Actor, q ReportQuery) (domain.ReportFilter, error) {
	pid := q.ProjectID
	if pid == nil {
		pid = a.ProjectID
	}
	if pid == nil {
		return domain.ReportFilter{}, apperror.Invalid("project_id wajib diisi")
	}
	if !a.CanView(*pid) {
		return domain.ReportFilter{}, apperror.ErrForbidden
	}
	from, err1 := time.Parse("2006-01-02", q.From)
	to, err2 := time.Parse("2006-01-02", q.To)
	if !ymd.MatchString(q.From) || !ymd.MatchString(q.To) || err1 != nil || err2 != nil || to.Before(from) {
		return domain.ReportFilter{}, apperror.Invalid("date_from & date_to wajib YYYY-MM-DD dan date_to ≥ date_from")
	}
	if to.Sub(from) > 366*24*time.Hour {
		return domain.ReportFilter{}, apperror.Invalid("periode maksimal 366 hari")
	}
	return domain.ReportFilter{ProjectID: *pid, From: q.From, To: q.To, ShiftID: q.ShiftID, Codes: q.Codes}, nil
}

func (s *Service) WeighbridgeDaily(ctx context.Context, a access.Actor, q ReportQuery) ([]domain.DailyCell, error) {
	f, err := s.filter(a, q)
	if err != nil {
		return nil, err
	}
	return s.repo.WeighbridgeDaily(ctx, f)
}

// ExecutiveHauling: produksi per customer pada `date` (harian), bulan berjalan (MTD), dan tahun berjalan (YTD).
func (s *Service) ExecutiveHauling(ctx context.Context, a access.Actor, projectID *uuid.UUID, date string) ([]domain.ExecutiveRow, error) {
	pid := projectID
	if pid == nil {
		pid = a.ProjectID
	}
	if pid == nil {
		return nil, apperror.Invalid("project_id wajib diisi")
	}
	if !a.CanView(*pid) {
		return nil, apperror.ErrForbidden
	}
	if _, err := time.Parse("2006-01-02", date); !ymd.MatchString(date) || err != nil {
		return nil, apperror.Invalid("date wajib YYYY-MM-DD")
	}
	return s.repo.ExecutiveHauling(ctx, *pid, date)
}

// ExecutiveEmplacement: stok emplacement project yang dilihat (project_id, atau project aktif) pada `date` (harian), MTD, dan YTD.
func (s *Service) ExecutiveEmplacement(ctx context.Context, a access.Actor, projectID *uuid.UUID, date string) ([]domain.ExecutiveEmplacementRow, error) {
	pid := projectID
	if pid == nil {
		pid = a.ProjectID
	}
	if pid == nil {
		return nil, apperror.Invalid("project_id wajib diisi")
	}
	if !a.CanView(*pid) {
		return nil, apperror.ErrForbidden
	}
	if _, err := time.Parse("2006-01-02", date); !ymd.MatchString(date) || err != nil {
		return nil, apperror.Invalid("date wajib YYYY-MM-DD")
	}
	return s.repo.ExecutiveEmplacement(ctx, *pid, date)
}

func (s *Service) BreakdownDaily(ctx context.Context, a access.Actor, q ReportQuery) ([]domain.DailyCell, error) {
	f, err := s.filter(a, q)
	if err != nil {
		return nil, err
	}
	return s.repo.BreakdownDaily(ctx, f)
}
