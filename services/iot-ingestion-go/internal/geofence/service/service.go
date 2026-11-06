// Package service = logika fitur geofence (saat ini hanya meneruskan filter ke repository).
package service

import (
	"context"

	"simha/iot-ingestion/internal/geofence/domain"
	"simha/iot-ingestion/internal/geofence/repository"
	"simha/iot-ingestion/internal/shared/database"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

// List = kejadian masuk/keluar area pada slot (tanggal operasional + shift), opsional satu segmen.
func (s *Service) List(ctx context.Context, slot database.Slot, segment *string) ([]domain.GeofenceRow, error) {
	return s.repo.GeofenceEvents(ctx, slot, segment)
}
