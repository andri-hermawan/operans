// Package service (fitur fuelcalibration) = kalibrasi fuel per equipment (B7, ADR 0008).
// Lihat = semua role project; ubah = admin project. Equipment dicek lewat fitur equipment.
package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	equipmentdomain "simha/core-api/internal/equipment/domain"
	"simha/core-api/internal/fuelcalibration/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.FuelCalibration, error)
	Get(ctx context.Context, equipmentID uuid.UUID) (*domain.FuelCalibration, error)
	Save(ctx context.Context, c domain.FuelCalibration, by uuid.UUID) (bool, error)
	Delete(ctx context.Context, equipmentID uuid.UUID) (bool, error)
}

// EquipmentGetter = fitur equipment: equipment yang boleh DILIHAT pemanggil (lainnya ErrNotFound).
type EquipmentGetter interface {
	Get(ctx context.Context, a access.Actor, id uuid.UUID) (*equipmentdomain.Equipment, error)
}

type Service struct {
	repo      Repository
	equipment EquipmentGetter
}

func New(repo Repository, equipment EquipmentGetter) *Service {
	return &Service{repo: repo, equipment: equipment}
}

// FuelCalibrationInput = isian kalibrasi. Tanpa points → titik dibangkitkan tiap 5 liter (seperti lama).
type FuelCalibrationInput struct {
	CalibrationType    string                    `json:"calibration_type"`     // lls | percentage
	TankCapacityLiters *float64                  `json:"tank_capacity_liters"` // wajib bila tanpa points
	MaxSensorValue     *float64                  `json:"max_sensor_value"`     // lls: wajib bila tanpa points; percentage: 100
	Points             []domain.CalibrationPoint `json:"points"`               // tabel uji isi tangki (opsional)
}

const maxCalibrationPoints = 500

func (s *Service) List(ctx context.Context, a access.Actor, projectID *uuid.UUID) ([]domain.FuelCalibration, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, pid)
}

func (s *Service) Get(ctx context.Context, a access.Actor, equipmentID uuid.UUID) (*domain.FuelCalibration, error) {
	if _, err := s.equipmentFor(ctx, a, equipmentID, false); err != nil {
		return nil, err
	}
	c, err := s.repo.Get(ctx, equipmentID)
	if err == nil && c == nil {
		return nil, fmt.Errorf("%w: equipment ini belum punya kalibrasi fuel", apperror.ErrNotFound)
	}
	return c, err
}

// Set mengganti seluruh kalibrasi equipment. created=true bila sebelumnya belum ada.
func (s *Service) Set(ctx context.Context, a access.Actor, equipmentID uuid.UUID, in FuelCalibrationInput) (*domain.FuelCalibration, bool, error) {
	if _, err := s.equipmentFor(ctx, a, equipmentID, true); err != nil {
		return nil, false, err
	}
	c, err := buildCalibration(in)
	if err != nil {
		return nil, false, err
	}
	c.EquipmentID = equipmentID
	created, err := s.repo.Save(ctx, *c, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, equipmentID)
	return got, created, err
}

func (s *Service) Delete(ctx context.Context, a access.Actor, equipmentID uuid.UUID) error {
	if _, err := s.equipmentFor(ctx, a, equipmentID, true); err != nil {
		return err
	}
	found, err := s.repo.Delete(ctx, equipmentID)
	if err == nil && !found {
		return fmt.Errorf("%w: equipment ini belum punya kalibrasi fuel", apperror.ErrNotFound)
	}
	return err
}

// FuelReading = hasil konversi nilai sensor (untuk cek kalibrasi; IoT memakai rumus yang sama di Fase C).
type FuelReading struct {
	SensorValue  float64  `json:"sensor_value"`
	VolumeLiters *float64 `json:"volume_liters"` // nil = di luar rentang kalibrasi
	Percent      *float64 `json:"percent"`
}

func (s *Service) Lookup(ctx context.Context, a access.Actor, equipmentID uuid.UUID, value float64) (*FuelReading, error) {
	c, err := s.Get(ctx, a, equipmentID)
	if err != nil {
		return nil, err
	}
	r := &FuelReading{SensorValue: value}
	if l, ok := domain.Liters(c.Points, value); ok {
		l = validate.Round2(l)
		p := validate.Round2(l / c.TankCapacityLiters * 100)
		r.VolumeLiters, r.Percent = &l, &p
	}
	return r, nil
}

// buildCalibration memvalidasi isian menjadi kalibrasi lengkap (kepala + titik urut).
func buildCalibration(in FuelCalibrationInput) (*domain.FuelCalibration, error) {
	typ := strings.ToLower(strings.TrimSpace(in.CalibrationType))
	if typ == "" {
		typ = domain.CalibrationLLS // bawaan lama
	}
	if typ != domain.CalibrationLLS && typ != domain.CalibrationPercentage {
		return nil, apperror.Invalid("calibration_type harus lls atau percentage")
	}
	c := &domain.FuelCalibration{CalibrationType: typ}
	sensorLimit := 100000.0
	if typ == domain.CalibrationPercentage {
		sensorLimit = 100
	}

	if len(in.Points) == 0 { // titik dibangkitkan tiap 5 liter
		capacity, max := in.TankCapacityLiters, in.MaxSensorValue
		if typ == domain.CalibrationPercentage {
			hundred := 100.0
			if max != nil && *max != 100 {
				return nil, apperror.Invalid("max_sensor_value untuk percentage selalu 100")
			}
			max = &hundred
		}
		if capacity == nil || *capacity <= 0 || *capacity > 10000 {
			return nil, apperror.Invalid("tank_capacity_liters wajib diisi (lebih dari 0, maks. 10000)")
		}
		if max == nil || *max <= 0 || *max > sensorLimit {
			return nil, apperror.Invalid("max_sensor_value wajib diisi (nilai sensor saat tangki penuh, lebih dari 0)")
		}
		c.TankCapacityLiters, c.MaxSensorValue, c.IsLinear = validate.Round2(*capacity), validate.Round2(*max), true
		c.Points = domain.LinearPoints(c.TankCapacityLiters, c.MaxSensorValue, typ)
		return c, nil
	}

	// tabel titik sendiri (uji isi tangki)
	if len(in.Points) < 2 || len(in.Points) > maxCalibrationPoints {
		return nil, apperror.Invalid("points minimal 2 dan maksimal %d titik", maxCalibrationPoints)
	}
	pts := make([]domain.CalibrationPoint, len(in.Points))
	for i, p := range in.Points {
		if p.SensorValue < 0 || p.SensorValue > sensorLimit || p.VolumeLiters < 0 || p.VolumeLiters > 10000 ||
			math.IsNaN(p.SensorValue) || math.IsNaN(p.VolumeLiters) {
			return nil, apperror.Invalid("titik ke-%d di luar batas (sensor 0..%v, liter 0..10000)", i+1, sensorLimit)
		}
		pts[i] = domain.CalibrationPoint{SensorValue: validate.Round2(p.SensorValue), VolumeLiters: validate.Round2(p.VolumeLiters)}
	}
	sort.SliceStable(pts, func(i, j int) bool { return pts[i].SensorValue < pts[j].SensorValue })
	for i := 1; i < len(pts); i++ {
		if pts[i].SensorValue == pts[i-1].SensorValue {
			return nil, apperror.Invalid("nilai sensor %v muncul lebih dari sekali", pts[i].SensorValue)
		}
		if pts[i].VolumeLiters < pts[i-1].VolumeLiters {
			return nil, apperror.Invalid("liter harus ikut naik bila nilai sensor naik (sensor %v → %v L lebih kecil dari titik sebelumnya)",
				pts[i].SensorValue, pts[i].VolumeLiters)
		}
	}
	last := pts[len(pts)-1]
	if last.VolumeLiters <= 0 {
		return nil, apperror.Invalid("liter titik terakhir (tangki penuh) harus lebih dari 0")
	}
	if in.TankCapacityLiters != nil && validate.Round2(*in.TankCapacityLiters) != last.VolumeLiters {
		return nil, apperror.Invalid("tank_capacity_liters harus sama dengan liter titik terakhir (%v) atau dikosongkan", last.VolumeLiters)
	}
	if in.MaxSensorValue != nil && validate.Round2(*in.MaxSensorValue) != last.SensorValue {
		return nil, apperror.Invalid("max_sensor_value harus sama dengan nilai sensor titik terakhir (%v) atau dikosongkan", last.SensorValue)
	}
	c.TankCapacityLiters, c.MaxSensorValue, c.Points = last.VolumeLiters, last.SensorValue, pts
	return c, nil
}

// equipmentFor memastikan equipment ada & user berhak (write → admin project).
func (s *Service) equipmentFor(ctx context.Context, a access.Actor, id uuid.UUID, write bool) (*equipmentdomain.Equipment, error) {
	e, err := s.equipment.Get(ctx, a, id) // equipment project lain = ErrNotFound
	if err != nil {
		return nil, err
	}
	if write && !a.CanManageProject(e.ProjectID) {
		return nil, apperror.ErrForbidden
	}
	return e, nil
}
