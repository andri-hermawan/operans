// Package service (fitur weighbridge) = tiket timbangan (B5c). Net = gross − tare (ADR 0007).
// Lihat: semua role project; isi/ubah/hapus: admin & dispatcher. Project, shift & master equipment
// dimuat lewat dailyinput.Loader (service fitur project, shift, equipment, reference).
package service

import (
	"context"
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
	"simha/core-api/internal/weighbridge/domain"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.WeighbridgeTicket, error)
	Get(ctx context.Context, id int64) (*domain.WeighbridgeTicket, error)
	Upsert(ctx context.Context, w domain.WeighbridgeTicket, by uuid.UUID) (int64, bool, error)
	Update(ctx context.Context, w domain.WeighbridgeTicket, by uuid.UUID) error
	Delete(ctx context.Context, id int64) error
	BulkUpsert(ctx context.Context, rows []domain.WeighbridgeTicket, by uuid.UUID, dryRun bool) (int, int, error)
}

type Service struct {
	repo  Repository
	daily dailyinput.Loader
}

func New(repo Repository, daily dailyinput.Loader) *Service {
	return &Service{repo: repo, daily: daily}
}

// WeighbridgeInput = isian form tiket timbangan. Berat dalam kg. Pada ubah (PATCH), nil = tidak diubah.
type WeighbridgeInput struct {
	ProjectID       *uuid.UUID `json:"project_id"` // superadmin
	OperationalDate *string    `json:"operational_date"`
	Shift           *string    `json:"shift"`
	TicketNo        *string    `json:"ticket_no"`
	EquipmentCode   *string    `json:"equipment_code"`
	Product         *string    `json:"product"`
	GrossKg         *float64   `json:"gross_kg"`
	TareKg          *float64   `json:"tare_kg"`
	NetKg           *float64   `json:"net_kg"` // kosong → gross − tare; diisi tetapi beda → ditolak
	Recipient       *string    `json:"recipient"`
	Customer        *string    `json:"customer"`
	Transporter     *string    `json:"transporter"`
	GrossTime       *string    `json:"gross_time"` // HH:MM jam lokal project
	TareTime        *string    `json:"tare_time"`
	GrossOperator   *string    `json:"gross_operator"`
	TareOperator    *string    `json:"tare_operator"`
	Description     *string    `json:"description"`
	Location        *string    `json:"location"`
}

func (s *Service) List(ctx context.Context, a access.Actor, f dailyinput.DailyFilter) ([]domain.WeighbridgeTicket, error) {
	if err := dailyinput.CheckFilter(a, &f); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

// Set: tanggal + nomor tiket sama → diperbarui.
func (s *Service) Set(ctx context.Context, a access.Actor, in WeighbridgeInput) (*domain.WeighbridgeTicket, bool, error) {
	pd, err := s.daily.Load(ctx, a, in.ProjectID, true)
	if err != nil {
		return nil, false, err
	}
	w := domain.WeighbridgeTicket{ProjectID: pd.Project.ID}
	if errs := applyWeighbridge(pd, &w, in); len(errs) > 0 {
		return nil, false, apperror.Invalid("%s", errs[0].Message)
	}
	id, created, err := s.repo.Upsert(ctx, w, a.UserID)
	if err != nil {
		return nil, false, err
	}
	got, err := s.repo.Get(ctx, id)
	return got, created, err
}

func (s *Service) Update(ctx context.Context, a access.Actor, id int64, patch WeighbridgeInput) (*domain.WeighbridgeTicket, error) {
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
	in := WeighbridgeInput{
		OperationalDate: &cur.OperationalDate, Shift: &cur.ShiftCode, TicketNo: &cur.TicketNo,
		EquipmentCode: &cur.EquipmentCode, Product: cur.Product, GrossKg: &cur.GrossKg, TareKg: &cur.TareKg,
		Recipient: cur.Recipient, Customer: cur.Customer, Transporter: cur.Transporter,
		GrossTime: dailyinput.ClockOf(cur.GrossAt, pd.Loc), TareTime: dailyinput.ClockOf(cur.TareAt, pd.Loc),
		GrossOperator: cur.GrossOperator, TareOperator: cur.TareOperator, Description: cur.Description, Location: cur.Location,
	} // net tidak disalin: selalu dihitung ulang dari gross − tare
	dailyinput.Overlay(&in, patch)
	if errs := applyWeighbridge(pd, cur, in); len(errs) > 0 {
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

// applyWeighbridge memvalidasi isian LENGKAP ke w. Net = gross − tare (ADR 0007).
// Jam timbang tidak dibatasi jam shift (truk bisa ditimbang sedikit lewat pergantian shift).
func applyWeighbridge(pd *dailyinput.ProjectData, w *domain.WeighbridgeTicket, in WeighbridgeInput) []dailyinput.RowIssue {
	var errs []dailyinput.RowIssue
	sh, dateOK := pd.ApplyDateShift(&w.OperationalDate, in.OperationalDate, in.Shift, &errs)
	w.ShiftID, w.ShiftCode, w.ShiftName = sh.ID, sh.Code, sh.Name
	if v, ok := dailyinput.Required(in.TicketNo, "ticket_no", &errs); ok {
		if utf8.RuneCountInString(v) > 50 {
			errs = append(errs, dailyinput.RowIssue{Column: "ticket_no", Message: "ticket_no maks. 50 karakter"})
		}
		w.TicketNo = v
	}
	if v, ok := dailyinput.Required(in.EquipmentCode, "equipment_code", &errs); ok {
		pd.SetEquipment(&w.EquipmentCode, &w.EquipmentID, v, &errs, "equipment_code")
	}
	w.Product = dailyinput.Optional(in.Product, "product", 50, &errs)
	w.Recipient = dailyinput.Optional(in.Recipient, "recipient", 100, &errs)
	w.Customer = dailyinput.Optional(in.Customer, "customer", 100, &errs)
	w.Transporter = dailyinput.Optional(in.Transporter, "transporter", 100, &errs)
	w.GrossOperator = dailyinput.Optional(in.GrossOperator, "gross_operator", 50, &errs)
	w.TareOperator = dailyinput.Optional(in.TareOperator, "tare_operator", 50, &errs)
	w.Description = dailyinput.Optional(in.Description, "description", 255, &errs)
	w.Location = dailyinput.Optional(in.Location, "location", 100, &errs)

	gross, okG := weight(in.GrossKg, "gross_kg", true, &errs)
	tare, okT := weight(in.TareKg, "tare_kg", true, &errs)
	net, okN := weight(in.NetKg, "net_kg", false, &errs)
	if okG && okT {
		w.GrossKg, w.TareKg, w.NetKg = gross, tare, validate.Round2(gross-tare)
		switch {
		case tare > gross:
			errs = append(errs, dailyinput.RowIssue{Column: "tare_kg", Message: "tare_kg lebih besar dari gross_kg"})
		case okN && in.NetKg != nil && math.Abs(net-w.NetKg) > 0.005:
			errs = append(errs, dailyinput.RowIssue{Column: "net_kg",
				Message: fmt.Sprintf("net_kg %v tidak sama dengan gross_kg − tare_kg (%v)", net, w.NetKg)})
		}
	}

	w.GrossAt, w.TareAt = nil, nil
	if dateOK {
		w.GrossAt = pd.ClockAt(sh, w.OperationalDate, in.GrossTime, "gross_time", &errs)
		w.TareAt = pd.ClockAt(sh, w.OperationalDate, in.TareTime, "tare_time", &errs)
	}
	return errs
}

// weight: berat kg ≥ 0, dibulatkan 2 desimal. ok=false bila kosong/salah.
func weight(v *float64, col string, need bool, errs *[]dailyinput.RowIssue) (float64, bool) {
	switch {
	case v == nil:
		if need {
			*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " wajib diisi"})
		}
		return 0, false
	case math.IsNaN(*v): // sudah dilaporkan pengurai Excel
		return 0, false
	case *v < 0 || *v >= 1e10:
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " harus 0 atau lebih (kg)"})
		return 0, false
	}
	return validate.Round2(*v), true
}
