// Package service = logika fitur alert (saat ini hanya meneruskan filter ke repository).
package service

import (
	"context"

	"simha/iot-ingestion/internal/alert/domain"
	"simha/iot-ingestion/internal/alert/repository"
	"simha/iot-ingestion/internal/shared/database"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

// List = alert dalam rentang tanggal operasional (+ shift/jenis/unit opsional).
func (s *Service) List(ctx context.Context, f database.RangeFilter) ([]domain.AlertRow, error) {
	return s.repo.Alerts(ctx, f)
}
