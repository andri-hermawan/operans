// Package service (fitur fuelfilling) = Fuel Filling: pengisian BBM unit per tanggal + shift (ADR 0019).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Total jam = HM pengisian − HM sebelumnya dan
// konsumsi = pemakaian BBM / total jam, keduanya hanya dihitung bila dikosongkan.
package service

import (
	"context"

	"github.com/google/uuid"

	"simha/core-api/internal/fuelfilling/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.FuelFilling, error)
	Get(ctx context.Context, id int64) (*domain.FuelFilling, error)
	Upsert(ctx context.Context, f domain.FuelFilling, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, f domain.FuelFilling, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.FuelFilling, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// FuelFillingInput = isian form Fuel Filling. HM dalam jam, BBM dalam liter. Pada ubah (PATCH), nil = tidak diubah.
type FuelFillingInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"`
	EquipmentCode   *string    `json:"equipment_code"`
	EquipmentModel  *string    `json:"equipment_model"`
	HmPrevious      *float64   `json:"hm_previous"`
	HmFilling       *float64   `json:"hm_filling"`
	TotalHours      *float64   `json:"total_hours"`          // kosong → HM pengisian − HM sebelumnya
	FuelUsage       *float64   `json:"fuel_usage_liter"`     //
	FuelConsumption *float64   `json:"fuel_consumption_lph"` // kosong → pemakaian / total jam
	WasteRemoval    *float64   `json:"waste_removal_liter"`
	CoalMining      *float64   `json:"coal_mining_liter"`
	Hauling         *float64   `json:"hauling_liter"`
	Rental          *float64   `json:"rental_liter"`
	Dewatering      *float64   `json:"dewatering_liter"`
	FinalSupport    *float64   `json:"final_support_liter"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.FuelFilling, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + shift + unit + HM pengisian sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in FuelFillingInput) (*domain.FuelFilling, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	f := domain.FuelFilling{ProjectID: pd.Project.ID}
	if errs := applyFuelFilling(pd, &f, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, f, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch FuelFillingInput) (*domain.FuelFilling, error) {
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil || !a.CanView(cur.ProjectID) {
		return nil, apperror.ErrNotFound
	}
	pd, err := s.daily.Load(ctx, a, &cur.ProjectID, true)
	if err != nil {
		return nil, err
	}
	in := FuelFillingInput{OperationalDate: &cur.OperationalDate, Shift: &cur.ShiftCode, EquipmentCode: &cur.EquipmentCode,
		EquipmentModel: cur.EquipmentModel, HmPrevious: cur.HmPrevious, HmFilling: cur.HmFilling, TotalHours: cur.TotalHours,
		FuelUsage: cur.FuelUsage, FuelConsumption: cur.FuelConsumption, WasteRemoval: cur.WasteRemoval, CoalMining: cur.CoalMining,
		Hauling: cur.Hauling, Rental: cur.Rental, Dewatering: cur.Dewatering, FinalSupport: cur.FinalSupport}
	if (patch.HmPrevious != nil || patch.HmFilling != nil) && patch.TotalHours == nil {
		in.TotalHours = nil // HM berubah tanpa total baru → total jam dihitung ulang
	}
	if (patch.HmPrevious != nil || patch.HmFilling != nil || patch.TotalHours != nil || patch.FuelUsage != nil) && patch.FuelConsumption == nil {
		in.FuelConsumption = nil // dasar hitung berubah tanpa konsumsi baru → dihitung ulang
	}
	dailyinput.Overlay(&in, patch)
	if errs := applyFuelFilling(pd, cur, in); len(errs) > 0 {
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

// applyFuelFilling memvalidasi isian LENGKAP ke f: tanggal, shift, unit wajib; semua angka ≥ 0 (2 desimal);
// HM pengisian ≥ HM sebelumnya. Total jam dan konsumsi dihitung bila kosong.
func applyFuelFilling(pd *dailyinput.ProjectData, f *domain.FuelFilling, in FuelFillingInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, _ := pd.ApplyDateShift(&f.OperationalDate, in.OperationalDate, in.Shift, &errs)
	f.ShiftID, f.ShiftCode, f.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&f.EquipmentCode, &f.EquipmentID, v, &errs, "equipment_code")
	}
	f.EquipmentModel = dailyinput.Optional(in.EquipmentModel, "equipment_model", 50, &errs)

	amount := func(v *float64, col string) *float64 { return dailyinput.Amount(v, col, 2, &errs) }
	f.HmPrevious = amount(in.HmPrevious, "hm_previous")
	f.HmFilling = amount(in.HmFilling, "hm_filling")
	f.TotalHours = amount(in.TotalHours, "total_hours")
	f.FuelUsage = amount(in.FuelUsage, "fuel_usage_liter")
	f.FuelConsumption = amount(in.FuelConsumption, "fuel_consumption_lph")
	f.WasteRemoval = amount(in.WasteRemoval, "waste_removal_liter")
	f.CoalMining = amount(in.CoalMining, "coal_mining_liter")
	f.Hauling = amount(in.Hauling, "hauling_liter")
	f.Rental = amount(in.Rental, "rental_liter")
	f.Dewatering = amount(in.Dewatering, "dewatering_liter")
	f.FinalSupport = amount(in.FinalSupport, "final_support_liter")

	if f.HmPrevious != nil && f.HmFilling != nil {
		if *f.HmFilling < *f.HmPrevious {
			errs = append(errs, dailyinput.RowIssue{Column: "hm_filling", Message: "hm_filling lebih kecil dari hm_previous"})
		} else if f.TotalHours == nil {
			total := validate.Round2(*f.HmFilling - *f.HmPrevious)
			f.TotalHours = &total
		}
	}
	if f.FuelConsumption == nil && f.FuelUsage != nil && f.TotalHours != nil && *f.TotalHours > 0 {
		c := validate.Round2(*f.FuelUsage / *f.TotalHours)
		f.FuelConsumption = &c
	}
	return errs
}
