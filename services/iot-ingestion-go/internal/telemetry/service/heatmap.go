package service

import (
	"context"

	"simha/iot-ingestion/internal/shared/database"
	"simha/iot-ingestion/internal/telemetry/domain"
)

// HeatGrid = kotak-kotak ringkasan kecepatan untuk Speed Heatmap (lihat repository.HeatGrid).
func (s *Service) HeatGrid(ctx context.Context, slot database.Slot, q domain.HeatGridQuery) ([]domain.HeatCell, error) {
	return s.repo.HeatGrid(ctx, slot, q)
}
