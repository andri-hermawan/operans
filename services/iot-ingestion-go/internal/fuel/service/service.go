// Package service = logika fitur fuel (saat ini hanya meneruskan filter ke repository).
package service

import (
	"context"

	"simha/iot-ingestion/internal/fuel/domain"
	"simha/iot-ingestion/internal/fuel/repository"
	"simha/iot-ingestion/internal/shared/database"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

// List = kejadian fuel dalam rentang tanggal operasional (laporan Fuel History).
func (s *Service) List(ctx context.Context, f database.RangeFilter) ([]domain.FuelEventRow, error) {
	return s.repo.FuelEventsRange(ctx, f)
}
