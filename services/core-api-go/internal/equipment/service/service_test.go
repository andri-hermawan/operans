package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"simha/core-api/internal/equipment/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
)

var (
	projCHR = uuid.New()
	projCMN = uuid.New()
)

func ptr[T any](v T) *T { return &v }

// eqMem = database palsu untuk equipment.
type eqMem struct {
	equipment map[uuid.UUID]*domain.Equipment
}

func newEqMem() *eqMem {
	return &eqMem{equipment: map[uuid.UUID]*domain.Equipment{}}
}

func (m *eqMem) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	if id == projCHR || id == projCMN {
		return &projectdomain.Project{ID: id}, nil
	}
	return nil, nil
}
func (m *eqMem) List(_ context.Context, p *uuid.UUID, _ string, inactive bool) ([]domain.Equipment, error) {
	out := []domain.Equipment{}
	for _, e := range m.equipment {
		if (p == nil || e.ProjectID == *p) && (inactive || e.IsActive) {
			out = append(out, *e)
		}
	}
	return out, nil
}
func (m *eqMem) Get(_ context.Context, id uuid.UUID) (*domain.Equipment, error) {
	if e, ok := m.equipment[id]; ok {
		c := *e
		return &c, nil
	}
	return nil, nil
}
func (m *eqMem) Insert(_ context.Context, e domain.Equipment, _ uuid.UUID) (uuid.UUID, error) {
	for _, x := range m.equipment {
		if x.ProjectID == e.ProjectID && x.EquipmentCode == e.EquipmentCode {
			return uuid.Nil, apperror.ErrCodeExists
		}
	}
	e.ID = uuid.New()
	m.equipment[e.ID] = &e
	return e.ID, nil
}
func (m *eqMem) Update(_ context.Context, id uuid.UUID, in domain.EquipmentInput, _ uuid.UUID) error {
	e := m.equipment[id]
	if in.EquipmentCode != nil {
		e.EquipmentCode = *in.EquipmentCode
	}
	if in.IsActive != nil {
		e.IsActive = *in.IsActive
	}
	return nil
}
func (m *eqMem) CodeMap(context.Context, uuid.UUID) (map[string]uuid.UUID, error) { return nil, nil }

// Bagian aktivasi dari test ini pindah ke activation/service/service_test.go.
func TestEquipment(t *testing.T) {
	ctx := context.Background()
	st := newEqMem()
	s := New(st, st)
	sa := access.Actor{UserID: uuid.New(), SuperAdmin: true}
	admin := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleAdmin}
	disp := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleDispatcher}
	viewer := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleViewer}
	adminCMN := access.Actor{UserID: uuid.New(), ProjectID: &projCMN, Role: access.RoleAdmin}

	var dt *domain.Equipment
	t.Run("admin tambah equipment: kode dirapikan jadi huruf besar, masuk project-nya", func(t *testing.T) {
		var err error
		dt, err = s.Create(ctx, admin, domain.EquipmentInput{EquipmentCode: ptr(" dt10300 "), Brand: ptr("Hino"), ProjectID: &projCMN})
		if err != nil {
			t.Fatal(err)
		}
		if dt.EquipmentCode != "DT10300" || dt.ProjectID != projCHR || !dt.IsActive {
			t.Fatalf("hasil: %+v", dt)
		}
		if _, err := s.Create(ctx, admin, domain.EquipmentInput{EquipmentCode: ptr("DT10300")}); !errors.Is(err, apperror.ErrCodeExists) {
			t.Fatalf("kode ganda: err = %v", err)
		}
	})

	t.Run("dispatcher & viewer tidak boleh menambah/mengubah equipment", func(t *testing.T) {
		if _, err := s.Create(ctx, disp, domain.EquipmentInput{EquipmentCode: ptr("DT1")}); err != apperror.ErrForbidden {
			t.Fatalf("dispatcher: err = %v", err)
		}
		if _, err := s.Update(ctx, viewer, dt.ID, domain.EquipmentInput{Brand: ptr("X")}); err != apperror.ErrForbidden {
			t.Fatalf("viewer: err = %v", err)
		}
	})

	t.Run("semua role project boleh melihat; project lain tidak", func(t *testing.T) {
		if l, _ := s.List(ctx, viewer, nil, "", false); len(l) != 1 {
			t.Fatalf("viewer CHR melihat %d", len(l))
		}
		if l, _ := s.List(ctx, adminCMN, nil, "", false); len(l) != 0 {
			t.Fatal("admin CMN tidak boleh melihat equipment CHR")
		}
		if _, err := s.Get(ctx, adminCMN, dt.ID); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("superadmin wajib menyebut project saat menambah", func(t *testing.T) {
		var ve *apperror.ValidationError
		if _, err := s.Create(ctx, sa, domain.EquipmentInput{EquipmentCode: ptr("EX01")}); !errors.As(err, &ve) {
			t.Fatalf("err = %v", err)
		}
		if _, err := s.Create(ctx, sa, domain.EquipmentInput{EquipmentCode: ptr("EX01"), ProjectID: &projCMN}); err != nil {
			t.Fatal(err)
		}
	})
}
