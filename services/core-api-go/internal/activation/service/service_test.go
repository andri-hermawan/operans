package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/activation/domain"
	equipmentdomain "simha/core-api/internal/equipment/domain"
	equipmentsvc "simha/core-api/internal/equipment/service"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

var (
	projCHR = uuid.New()
	projCMN = uuid.New()
)

func ptr[T any](v T) *T { return &v }

// eqMem = database palsu equipment (dipakai service equipment yang asli) + aktivasi.
type eqMem struct {
	equipmentsvc.Repository // method equipment lain tidak dipakai di test ini
	equipment               map[uuid.UUID]*equipmentdomain.Equipment
	activations             map[int64]*domain.Activation
	nextID                  int64
}

func newEqMem() *eqMem {
	return &eqMem{equipment: map[uuid.UUID]*equipmentdomain.Equipment{}, activations: map[int64]*domain.Activation{}}
}

// --- equipment (untuk equipmentsvc) ---

func (m *eqMem) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return &projectdomain.Project{ID: id}, nil
}
func (m *eqMem) Get(_ context.Context, id uuid.UUID) (*equipmentdomain.Equipment, error) {
	if e, ok := m.equipment[id]; ok {
		c := *e
		return &c, nil
	}
	return nil, nil
}

// --- aktivasi (untuk service ini) ---

type actMem struct{ *eqMem }

func (m actMem) List(_ context.Context, f domain.ActivationFilter) ([]domain.Activation, error) {
	out := []domain.Activation{}
	for _, a := range m.activations {
		if f.ProjectID == nil || a.ProjectID == *f.ProjectID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (m actMem) Get(_ context.Context, id int64) (*domain.Activation, error) {
	if a, ok := m.activations[id]; ok {
		c := *a
		return &c, nil
	}
	return nil, nil
}
func (m actMem) Upsert(_ context.Context, eq uuid.UUID, date, status string, desc *string, _ uuid.UUID) (int64, bool, error) {
	for _, a := range m.activations {
		if a.EquipmentID == eq && a.ActivationDate == date {
			a.Status, a.Description = status, desc
			return a.ID, false, nil
		}
	}
	m.nextID++
	e := m.equipment[eq]
	m.activations[m.nextID] = &domain.Activation{ID: m.nextID, EquipmentID: eq, EquipmentCode: e.EquipmentCode,
		ProjectID: e.ProjectID, ActivationDate: date, Status: status, Description: desc, UpdatedAt: time.Now()}
	return m.nextID, true, nil
}
func (m actMem) Update(_ context.Context, id int64, date, status, _ *string, _ uuid.UUID) error {
	if status != nil {
		m.activations[id].Status = *status
	}
	if date != nil {
		m.activations[id].ActivationDate = *date
	}
	return nil
}
func (m actMem) Delete(_ context.Context, id int64) error {
	delete(m.activations, id)
	return nil
}
func (m actMem) BulkUpsert(context.Context, []domain.ActivationRow, uuid.UUID, bool) (int, int, error) {
	return 0, 0, nil
}

// Dulu bagian dari TestEquipment (service/equipment_test.go).
func TestEquipmentActivation(t *testing.T) {
	ctx := context.Background()
	st := newEqMem()
	s := New(actMem{st}, equipmentsvc.New(st, st))
	disp := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleDispatcher}
	viewer := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleViewer}
	adminCMN := access.Actor{UserID: uuid.New(), ProjectID: &projCMN, Role: access.RoleAdmin}

	dt := &equipmentdomain.Equipment{ID: uuid.New(), ProjectID: projCHR, EquipmentCode: "DT10300", IsActive: true}
	st.equipment[dt.ID] = dt

	t.Run("aktivasi: dispatcher boleh isi; isi ulang tanggal sama = diperbarui; viewer ditolak", func(t *testing.T) {
		a1, created, err := s.Set(ctx, disp, ActivationInput{EquipmentID: &dt.ID, ActivationDate: ptr("2026-10-22"), Status: ptr("active")})
		if err != nil || !created {
			t.Fatalf("created=%v err=%v", created, err)
		}
		a2, created, err := s.Set(ctx, disp, ActivationInput{EquipmentID: &dt.ID, ActivationDate: ptr("2026-10-22"), Status: ptr("inactive")})
		if err != nil || created || a2.ID != a1.ID || a2.Status != "inactive" {
			t.Fatalf("seharusnya memperbarui baris yang sama: %+v created=%v err=%v", a2, created, err)
		}
		if _, _, err := s.Set(ctx, viewer, ActivationInput{EquipmentID: &dt.ID, ActivationDate: ptr("2026-10-23"), Status: ptr("active")}); err != apperror.ErrForbidden {
			t.Fatalf("viewer: err = %v", err)
		}
		var ve *apperror.ValidationError
		if _, _, err := s.Set(ctx, disp, ActivationInput{EquipmentID: &dt.ID, ActivationDate: ptr("22-10-2026"), Status: ptr("active")}); !errors.As(err, &ve) {
			t.Fatalf("format tanggal salah: err = %v", err)
		}
		if _, _, err := s.Set(ctx, disp, ActivationInput{EquipmentID: &dt.ID, ActivationDate: ptr("2026-10-24"), Status: ptr("Active")}); !errors.As(err, &ve) {
			t.Fatalf("status huruf besar (kode lama) harus ditolak: err = %v", err)
		}
		if err := s.Delete(ctx, adminCMN, a1.ID); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("admin project lain hapus: err = %v", err)
		}
		if err := s.Delete(ctx, disp, a1.ID); err != nil {
			t.Fatal(err)
		}
	})
}
