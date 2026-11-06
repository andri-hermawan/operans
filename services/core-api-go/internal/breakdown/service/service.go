// Package service (fitur breakdown) = Status Breakdown: catatan unit rusak/berhenti per tanggal + shift (B5b).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Project, shift, master equipment &
// daftar pilihan dimuat lewat dailyinput.Loader (service fitur project, shift, equipment, reference).
package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/breakdown/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.EquipmentBreakdown, error)
	Get(ctx context.Context, id int64) (*domain.EquipmentBreakdown, error)
	Upsert(ctx context.Context, b domain.EquipmentBreakdown, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, b domain.EquipmentBreakdown, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.EquipmentBreakdown, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// BreakdownInput = isian form breakdown. Pada ubah (PATCH), nil = tidak diubah; "" = dikosongkan.
type BreakdownInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"`
	EquipmentCode   *string    `json:"equipment_code"`
	Status          *string    `json:"status"`        // breakdown | continue | ready
	Category        *string    `json:"category"`      // kode atau label (reference_values)
	RepairStatus    *string    `json:"repair_status"` // kode atau label (reference_values)
	TimeStart       *string    `json:"time_start"`    // HH:MM jam lokal project
	TimeEnd         *string    `json:"time_end"`
	DurationHours   *float64   `json:"duration_hours"` // kosong → dihitung dari jam mulai/selesai
	Description     *string    `json:"description"`
	Location        *string    `json:"location"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.EquipmentBreakdown, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + shift + unit + kategori + jam mulai sama → diperbarui (seperti sistem lama).
func (s *Service) Set(ctx context.Context, a access.Actor, in BreakdownInput) (*domain.EquipmentBreakdown, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	b := domain.EquipmentBreakdown{ProjectID: pd.Project.ID}
	if errs := applyBreakdown(pd, &b, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, b, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch BreakdownInput) (*domain.EquipmentBreakdown, error) {
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
	in := breakdownToInput(cur, pd.Loc)
	if (patch.TimeStart != nil || patch.TimeEnd != nil) && patch.DurationHours == nil {
		in.DurationHours = nil // jam berubah → durasi dihitung ulang
	}
	dailyinput.Overlay(&in, patch)
	if errs := applyBreakdown(pd, cur, in); len(errs) > 0 {
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

// breakdownToInput mengubah data tersimpan kembali menjadi isian (untuk digabung dengan PATCH).
func breakdownToInput(b *domain.EquipmentBreakdown, loc *time.Location) BreakdownInput {
	in := BreakdownInput{
		OperationalDate: &b.OperationalDate, Shift: &b.ShiftCode, EquipmentCode: &b.EquipmentCode,
		Status: &b.Status, Category: &b.CategoryCode, RepairStatus: &b.RepairStatusCode,
		TimeStart: dailyinput.ClockOf(b.StartedAt, loc), TimeEnd: dailyinput.ClockOf(b.EndedAt, loc),
		Description: b.Description, Location: b.Location,
	}
	if b.StartedAt == nil || b.EndedAt == nil {
		in.DurationHours = b.DurationHours // durasi isian manual (tanpa jam lengkap)
	}
	return in
}

// applyBreakdown memvalidasi isian LENGKAP ke b. Aturan (ADR 0007):
//   - jam mulai/selesai harus di dalam jam shift; shift lewat tengah malam → jam setelah 00:00 = hari berikutnya
//   - status continue tanpa jam selesai → selesai = akhir shift
//   - durasi kosong → dihitung; diisi tetapi beda dengan jam selesai − mulai → ditolak
func applyBreakdown(pd *dailyinput.ProjectData, b *domain.EquipmentBreakdown, in BreakdownInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, dateOK := pd.ApplyDateShift(&b.OperationalDate, in.OperationalDate, in.Shift, &errs)
	b.ShiftID, b.ShiftCode, b.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&b.EquipmentCode, &b.EquipmentID, v, &errs, "equipment_code")
	}
	if v, ok := dailyinput.Required(in.Status, "status", &errs); ok {
		if st, ok := parseBreakdownStatus(v); ok {
			b.Status = st
		} else {
			errs = append(errs, dailyinput.RowIssue{Column: "status", Message: "status \"" + v + "\" tidak dikenal (Breakdown / Continue / Ready)"})
		}
	}
	if v, ok := dailyinput.Required(in.Category, "category", &errs); ok {
		b.CategoryCode = pd.RefCode(referencedomain.RefBreakdownCategory, v, "category", &errs)
	}
	if v, ok := dailyinput.Required(in.RepairStatus, "repair_status", &errs); ok {
		b.RepairStatusCode = pd.RefCode(referencedomain.RefRepairStatus, v, "repair_status", &errs)
	}
	b.Description = dailyinput.Optional(in.Description, "description", 255, &errs)
	b.Location = dailyinput.Optional(in.Location, "location", 100, &errs)

	b.StartedAt, b.EndedAt, b.DurationHours = nil, nil, nil
	if in.DurationHours != nil && !math.IsNaN(*in.DurationHours) { // NaN = sudah dilaporkan pengurai Excel
		if d := *in.DurationHours; d < 0 || d > 24 {
			errs = append(errs, dailyinput.RowIssue{Column: "duration_hours", Message: "duration_hours harus 0–24 jam"})
		} else {
			d = validate.Round2(d)
			b.DurationHours = &d
		}
	}
	if !dateOK {
		return errs
	}
	winStart, winEnd, err := sh.Window(b.OperationalDate, pd.Loc)
	if err != nil {
		return append(errs, dailyinput.RowIssue{Column: "shift", Message: err.Error()})
	}
	b.StartedAt = pd.ClockInShift(sh, b.OperationalDate, in.TimeStart, "time_start", winStart, winEnd, &errs)
	b.EndedAt = pd.ClockInShift(sh, b.OperationalDate, in.TimeEnd, "time_end", winStart, winEnd, &errs)
	if b.EndedAt == nil && b.StartedAt != nil && b.Status == domain.BreakdownStatusContinue {
		b.EndedAt = &winEnd
	}
	if b.StartedAt != nil && b.EndedAt != nil {
		if b.EndedAt.Before(*b.StartedAt) {
			return append(errs, dailyinput.RowIssue{Column: "time_end", Message: "time_end lebih awal dari time_start"})
		}
		d := validate.Round2(b.EndedAt.Sub(*b.StartedAt).Hours())
		if b.DurationHours != nil && math.Abs(*b.DurationHours-d) > 0.02 {
			errs = append(errs, dailyinput.RowIssue{Column: "duration_hours",
				Message: fmt.Sprintf("duration_hours %s tidak sama dengan time_end − time_start (%s)", dailyinput.HoursText(*b.DurationHours), dailyinput.HoursText(d))})
		}
		b.DurationHours = &d
	}
	return errs
}

func parseBreakdownStatus(s string) (string, bool) {
	switch dailyinput.NormRef(s) {
	case "breakdown", "down": // "down" = sebutan lama
		return domain.BreakdownStatusBreakdown, true
	case "continue":
		return domain.BreakdownStatusContinue, true
	case "ready":
		return domain.BreakdownStatusReady, true
	}
	return "", false
}
