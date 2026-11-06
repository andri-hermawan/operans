// Package service (fitur ritase) = Ritase: hasil angkut per jam kerja (ADR 0015).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Angka produksi disimpan apa adanya dari input.
package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"simha/core-api/internal/ritase/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.Ritase, error)
	Get(ctx context.Context, id int64) (*domain.Ritase, error)
	Upsert(ctx context.Context, t domain.Ritase, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, t domain.Ritase, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.Ritase, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// RitaseInput = isian form Ritase. Pada ubah (PATCH), nil = tidak diubah.
type RitaseInput struct {
	ProjectID          *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate    *string    `json:"operational_date"`
	Shift              *string    `json:"shift"`
	TimeSlot           *string    `json:"time_slot"` // 07.00-08.00
	LoaderCode         *string    `json:"loader_code"`
	LoaderModel        *string    `json:"loader_model"`
	LoaderOperatorNik  *string    `json:"loader_operator_nik"`
	LoaderOperatorName *string    `json:"loader_operator_name"`
	LoaderPartner      *string    `json:"loader_partner"`
	HaulerCode         *string    `json:"hauler_code"`
	HaulerModel        *string    `json:"hauler_model"`
	HaulerOperatorNik  *string    `json:"hauler_operator_nik"`
	HaulerOperatorName *string    `json:"hauler_operator_name"`
	HaulerPartner      *string    `json:"hauler_partner"`
	Block              *string    `json:"block"`
	Location           *string    `json:"location"`
	Dumping            *string    `json:"dumping"`
	Material           *string    `json:"material"`
	SubMaterial        *string    `json:"sub_material"`
	Seam               *string    `json:"seam"`
	Phase              *string    `json:"phase"`
	Distance           *float64   `json:"distance"`
	Ritase             *float64   `json:"ritase"`
	Production         *float64   `json:"production"`
	ProductionAdjusted *float64   `json:"production_adjusted"`
	DistanceVolume     *float64   `json:"distance_volume"`
	ProductivityFactor *float64   `json:"productivity_factor"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.Ritase, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: kombinasi kunci sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in RitaseInput) (*domain.Ritase, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	t := domain.Ritase{ProjectID: pd.Project.ID}
	if errs := applyRitase(pd, &t, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, t, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch RitaseInput) (*domain.Ritase, error) {
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
	in := ritaseToInput(cur)
	dailyinput.Overlay(&in, patch)
	if errs := applyRitase(pd, cur, in); len(errs) > 0 {
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

// ritaseToInput mengubah data tersimpan kembali menjadi isian (untuk digabung dengan PATCH).
func ritaseToInput(t *domain.Ritase) RitaseInput {
	return RitaseInput{OperationalDate: &t.OperationalDate, Shift: &t.ShiftCode, TimeSlot: &t.TimeSlot,
		LoaderCode: &t.LoaderCode, LoaderModel: t.LoaderModel, LoaderOperatorNik: t.LoaderOperatorNik,
		LoaderOperatorName: t.LoaderOperatorName, LoaderPartner: t.LoaderPartner,
		HaulerCode: &t.HaulerCode, HaulerModel: t.HaulerModel, HaulerOperatorNik: t.HaulerOperatorNik,
		HaulerOperatorName: t.HaulerOperatorName, HaulerPartner: t.HaulerPartner,
		Block: t.Block, Location: t.Location, Dumping: t.Dumping, Material: t.Material, SubMaterial: t.SubMaterial,
		Seam: t.Seam, Phase: t.Phase, Distance: t.Distance, Ritase: t.Ritase, Production: t.Production,
		ProductionAdjusted: t.ProductionAdjusted, DistanceVolume: t.DistanceVolume, ProductivityFactor: t.ProductivityFactor}
}

// applyRitase memvalidasi isian LENGKAP ke t: tanggal, shift, jam, loader & hauler wajib; angka ≥ 0.
func applyRitase(pd *dailyinput.ProjectData, t *domain.Ritase, in RitaseInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, _ := pd.ApplyDateShift(&t.OperationalDate, in.OperationalDate, in.Shift, &errs)
	t.ShiftID, t.ShiftCode, t.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.TimeSlot, "time_slot", &errs); ok {
		if slot, ok := NormalizeTimeSlot(v); ok {
			t.TimeSlot = slot
		} else {
			errs = append(errs, dailyinput.RowIssue{Column: "time_slot", Message: "time_slot \"" + v + "\" harus berformat 07.00-08.00"})
		}
	}
	if v, ok := dailyinput.Required(in.LoaderCode, "loader_code", &errs); ok {
		pd.SetEquipment(&t.LoaderCode, &t.LoaderEquipmentID, v, &errs, "loader_code")
	}
	if v, ok := dailyinput.Required(in.HaulerCode, "hauler_code", &errs); ok {
		pd.SetEquipment(&t.HaulerCode, &t.HaulerEquipmentID, v, &errs, "hauler_code")
	}
	t.LoaderModel = dailyinput.Optional(in.LoaderModel, "loader_model", 50, &errs)
	t.LoaderOperatorNik = dailyinput.Optional(in.LoaderOperatorNik, "loader_operator_nik", 30, &errs)
	t.LoaderOperatorName = dailyinput.Optional(in.LoaderOperatorName, "loader_operator_name", 100, &errs)
	t.LoaderPartner = dailyinput.Optional(in.LoaderPartner, "loader_partner", 50, &errs)
	t.HaulerModel = dailyinput.Optional(in.HaulerModel, "hauler_model", 50, &errs)
	t.HaulerOperatorNik = dailyinput.Optional(in.HaulerOperatorNik, "hauler_operator_nik", 30, &errs)
	t.HaulerOperatorName = dailyinput.Optional(in.HaulerOperatorName, "hauler_operator_name", 100, &errs)
	t.HaulerPartner = dailyinput.Optional(in.HaulerPartner, "hauler_partner", 50, &errs)
	t.Block = dailyinput.Optional(in.Block, "block", 50, &errs)
	t.Location = dailyinput.Optional(in.Location, "location", 100, &errs)
	t.Dumping = dailyinput.Optional(in.Dumping, "dumping", 100, &errs)
	t.Material = dailyinput.Optional(in.Material, "material", 50, &errs)
	t.SubMaterial = dailyinput.Optional(in.SubMaterial, "sub_material", 50, &errs)
	t.Seam = dailyinput.Optional(in.Seam, "seam", 50, &errs)
	t.Phase = dailyinput.Optional(in.Phase, "phase", 20, &errs)

	t.Distance = dailyinput.Amount(in.Distance, "distance", 2, &errs)
	t.Ritase = dailyinput.Amount(in.Ritase, "ritase", 2, &errs)
	t.Production = dailyinput.Amount(in.Production, "production", 2, &errs)
	t.ProductionAdjusted = dailyinput.Amount(in.ProductionAdjusted, "production_adjusted", 2, &errs)
	t.DistanceVolume = dailyinput.Amount(in.DistanceVolume, "distance_volume", 2, &errs)
	t.ProductivityFactor = dailyinput.Amount(in.ProductivityFactor, "productivity_factor", 4, &errs)
	return errs
}

var timeSlotRe = regexp.MustCompile(`^(\d{1,2})[.:](\d{2})\s*-\s*(\d{1,2})[.:](\d{2})$`)

// NormalizeTimeSlot: "7:00 - 8:00" / "07.00-08.00" → "07.00-08.00". Jam 0–24, menit 0–59.
func NormalizeTimeSlot(s string) (string, bool) {
	m := timeSlotRe.FindStringSubmatch(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
	if m == nil {
		return "", false
	}
	var h1, n1, h2, n2 int
	fmt.Sscanf(m[1]+" "+m[2]+" "+m[3]+" "+m[4], "%d %d %d %d", &h1, &n1, &h2, &n2)
	if h1 > 24 || h2 > 24 || n1 > 59 || n2 > 59 {
		return "", false
	}
	return fmt.Sprintf("%02d.%02d-%02d.%02d", h1, n1, h2, n2), true
}
