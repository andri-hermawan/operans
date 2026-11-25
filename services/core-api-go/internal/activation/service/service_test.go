package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/activation/domain"
	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	shiftdomain "simha/core-api/internal/shift/domain"
)

func ptr[T any](v T) *T { return &v }

// ---------- database palsu ----------

type fakeRepo struct {
	rows map[int64]*domain.Activation
	next int64
}

func (r *fakeRepo) List(_ context.Context, f domain.ActivationFilter) ([]domain.Activation, error) {
	out := []domain.Activation{}
	for _, a := range r.rows {
		if f.ProjectID == nil || a.ProjectID == *f.ProjectID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(_ context.Context, id int64) (*domain.Activation, error) {
	if a, ok := r.rows[id]; ok {
		c := *a
		return &c, nil
	}
	return nil, nil
}
func (r *fakeRepo) Upsert(_ context.Context, row domain.ActivationRow, _ uuid.UUID) (int64, bool, error) {
	for _, a := range r.rows {
		if a.ProjectID == row.ProjectID && a.EquipmentCode == row.EquipmentCode && a.ActivationDate == row.Date {
			a.Status, a.Description, a.EquipmentID = row.Status, row.Description, row.EquipmentID
			return a.ID, false, nil
		}
	}
	r.next++
	r.rows[r.next] = &domain.Activation{ID: r.next, ProjectID: row.ProjectID, EquipmentID: row.EquipmentID, EquipmentCode: row.EquipmentCode,
		ActivationDate: row.Date, Status: row.Status, Description: row.Description, UpdatedAt: time.Now()}
	return r.next, true, nil
}
func (r *fakeRepo) Update(_ context.Context, id int64, date, status, _ *string, _ uuid.UUID) error {
	if status != nil {
		r.rows[id].Status = *status
	}
	if date != nil {
		r.rows[id].ActivationDate = *date
	}
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, id int64) error { delete(r.rows, id); return nil }
func (r *fakeRepo) BulkUpsert(context.Context, []domain.ActivationRow, uuid.UUID, bool) (int, int, error) {
	return 0, 0, nil
}

type fakeProjects struct{}

func (fakeProjects) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return &projectdomain.Project{ID: id, Timezone: "Asia/Jakarta"}, nil
}

type noShifts struct{}

func (noShifts) ListActive(context.Context, uuid.UUID) ([]shiftdomain.Shift, error) { return nil, nil }

// masterEq = master equipment palsu: hanya DT10300 yang terdaftar.
var masterID = uuid.New()

type masterEq struct{}

func (masterEq) CodeMap(context.Context, uuid.UUID) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{"DT10300": masterID}, nil
}

type noRefs struct{}

func (noRefs) List(context.Context, string) ([]referencedomain.ReferenceValue, error) { return nil, nil }

func newService() *Service {
	loader := dailyinput.Loader{Projects: fakeProjects{}, Shifts: noShifts{}, Equipment: masterEq{}, References: noRefs{}}
	return New(&fakeRepo{rows: map[int64]*domain.Activation{}}, loader)
}

func actorOf(role string, pid uuid.UUID) access.Actor {
	return access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: role}
}

func TestEquipmentActivation(t *testing.T) {
	ctx := context.Background()
	s := newService()
	chr, cmn := uuid.New(), uuid.New()
	disp := actorOf(access.RoleDispatcher, chr)
	viewer := actorOf(access.RoleViewer, chr)
	adminCMN := actorOf(access.RoleAdmin, cmn)
	in := func(code, date, status string) ActivationInput {
		return ActivationInput{EquipmentCode: ptr(code), ActivationDate: ptr(date), Status: ptr(status)}
	}

	t.Run("dispatcher boleh isi; isi ulang tanggal sama = diperbarui; viewer ditolak", func(t *testing.T) {
		a1, created, err := s.Set(ctx, disp, in("DT10300", "2026-10-22", "active"))
		if err != nil || !created || a1.EquipmentID == nil || *a1.EquipmentID != masterID || a1.ProjectID != chr {
			t.Fatalf("created=%v err=%v %+v", created, err, a1)
		}
		a2, created, err := s.Set(ctx, disp, in("dt10300", "2026-10-22", "inactive")) // huruf kecil dinormalkan
		if err != nil || created || a2.ID != a1.ID || a2.Status != "inactive" {
			t.Fatalf("seharusnya memperbarui baris yang sama: %+v created=%v err=%v", a2, created, err)
		}
		if _, _, err := s.Set(ctx, viewer, in("DT10300", "2026-10-23", "active")); !errors.Is(err, apperror.ErrForbidden) {
			t.Fatalf("viewer: err = %v", err)
		}
		var ve *apperror.ValidationError
		if _, _, err := s.Set(ctx, disp, in("DT10300", "22-10-2026", "active")); !errors.As(err, &ve) {
			t.Fatalf("format tanggal salah: err = %v", err)
		}
		if _, _, err := s.Set(ctx, disp, in("DT10300", "2026-10-24", "Active")); !errors.As(err, &ve) {
			t.Fatalf("status huruf besar (kode lama) harus ditolak: err = %v", err)
		}
		if err := s.Delete(ctx, adminCMN, a1.ID); !errors.Is(err, apperror.ErrNotFound) {
			t.Fatalf("admin project lain hapus: err = %v", err)
		}
		if err := s.Delete(ctx, disp, a1.ID); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unit yang belum ada di master tetap tersimpan (equipment_id kosong)", func(t *testing.T) {
		a, created, err := s.Set(ctx, disp, in("WT99999", "2026-10-25", "inactive"))
		if err != nil || !created || a.EquipmentCode != "WT99999" || a.EquipmentID != nil {
			t.Fatalf("unit non-master harus diterima: %+v created=%v err=%v", a, created, err)
		}
	})

	t.Run("kode unit wajib; unit tidak bisa diganti saat ubah", func(t *testing.T) {
		var ve *apperror.ValidationError
		if _, _, err := s.Set(ctx, disp, ActivationInput{ActivationDate: ptr("2026-10-26"), Status: ptr("active")}); !errors.As(err, &ve) {
			t.Fatalf("tanpa kode unit: err = %v", err)
		}
		a, _, _ := s.Set(ctx, disp, in("DT10300", "2026-10-27", "active"))
		if _, err := s.Update(ctx, disp, a.ID, ActivationInput{EquipmentCode: ptr("DT10301")}); !errors.As(err, &ve) {
			t.Fatalf("ganti unit: err = %v", err)
		}
		if got, err := s.Update(ctx, disp, a.ID, ActivationInput{Status: ptr("inactive")}); err != nil || got.Status != "inactive" {
			t.Fatalf("ubah status: %+v err=%v", got, err)
		}
	})
}
