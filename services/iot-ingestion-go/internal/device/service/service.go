// Package service = logika fitur Device. Dipakai handler HTTP Device, dan oleh fitur ingest
// (lewat interface di sisi ingest) untuk mengenali alat dari IMEI (TCP) atau API key (HTTP).
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/device/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service { return &Service{repo: repo} }

// FindDeviceByIMEI: alat aktif dengan IMEI ini; (nil, nil) bila tidak terdaftar / nonaktif.
func (s *Service) FindDeviceByIMEI(ctx context.Context, imei string) (*domain.Device, error) {
	return s.repo.FindDeviceByIMEI(ctx, imei)
}

// FindDeviceByAPIKey: alat aktif pemilik API key ini (yang dicari = hash-nya); (nil, nil) bila tidak dikenal.
func (s *Service) FindDeviceByAPIKey(ctx context.Context, key string) (*domain.Device, error) {
	return s.repo.FindDeviceByKeyHash(ctx, domain.HashKey(key))
}

func (s *Service) List(ctx context.Context, projectID *uuid.UUID) ([]domain.DeviceRow, error) {
	return s.repo.ListDevices(ctx, projectID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.DeviceRow, error) {
	return s.repo.GetDevice(ctx, id)
}

func (s *Service) EquipmentProject(ctx context.Context, equipmentID uuid.UUID) (*uuid.UUID, error) {
	return s.repo.EquipmentProject(ctx, equipmentID)
}

func (s *Service) DeviceOnEquipment(ctx context.Context, equipmentID uuid.UUID, except *uuid.UUID) (string, error) {
	return s.repo.DeviceOnEquipment(ctx, equipmentID, except)
}

func (s *Service) Insert(ctx context.Context, in domain.DeviceInput) (uuid.UUID, error) {
	return s.repo.InsertDevice(ctx, in)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in domain.DeviceInput) error {
	return s.repo.UpdateDevice(ctx, id, in)
}

// SetAPIKey menyimpan HASH key baru (key aslinya tidak pernah disimpan).
func (s *Service) SetAPIKey(ctx context.Context, id uuid.UUID, key string) error {
	return s.repo.SetDeviceKeyHash(ctx, id, domain.HashKey(key))
}

func (s *Service) LastEvent(ctx context.Context, projectID *uuid.UUID, deviceID uuid.UUID) (eventTime, receivedAt *time.Time, err error) {
	return s.repo.DeviceLastEvent(ctx, projectID, deviceID)
}
