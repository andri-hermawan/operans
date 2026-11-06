// Package service = logika fitur telemetry (saat ini hanya meneruskan ke repository; batas rentang &
// jumlah baris sudah diperiksa handler sebelum sampai ke sini).
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/shared/database"
	"simha/iot-ingestion/internal/telemetry/domain"
	"simha/iot-ingestion/internal/telemetry/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

// Latest = data paling baru satu equipment (nil bila belum ada / milik project lain).
func (s *Service) Latest(ctx context.Context, projectID *uuid.UUID, equipmentID uuid.UUID) (*domain.TelemetryRow, error) {
	return s.repo.LatestByEquipment(ctx, projectID, equipmentID)
}

// History = riwayat satu equipment dalam rentang [from, to), terbaru dulu, maksimal limit baris.
func (s *Service) History(ctx context.Context, projectID *uuid.UUID, equipmentID uuid.UUID, from, to time.Time, limit int) ([]domain.TelemetryRow, error) {
	return s.repo.HistoryByEquipment(ctx, projectID, equipmentID, from, to, limit)
}

func (s *Service) Points(ctx context.Context, slot database.Slot, stepSec, limit int) ([]domain.Point, error) {
	return s.repo.Points(ctx, slot, stepSec, limit)
}

func (s *Service) SegmentSpeeds(ctx context.Context, slot database.Slot) ([]domain.SegmentSpeed, error) {
	return s.repo.SegmentSpeeds(ctx, slot)
}

func (s *Service) Activities(ctx context.Context, slot database.Slot) ([]domain.Activity, error) {
	return s.repo.Activities(ctx, slot)
}
