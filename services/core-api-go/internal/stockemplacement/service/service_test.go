package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	shiftdomain "simha/core-api/internal/shift/domain"
	"simha/core-api/internal/stockemplacement/domain"
)

func f(v float64) *float64 { return &v }

func TestApplyStock(t *testing.T) {
	var e domain.StockEmplacement
	in := StockEmplacementInput{Partner: dailyinput.PtrStr("CPM"), OperationalDate: dailyinput.PtrStr("01/10/2026"),
		Info: dailyinput.PtrStr("OK"), StockInitial: f(120000.456), TotalTlsOut: f(15000)}
	if errs := applyStock(&e, in); len(errs) != 0 {
		t.Fatalf("seharusnya valid: %v", errs)
	}
	if e.Partner != "CPM" || e.OperationalDate != "2026-10-01" || *e.StockInitial != 120000.46 || *e.TotalTlsOut != 15000 || e.Shift1 != nil {
		t.Errorf("hasil salah: %+v", e)
	}
}

func TestApplyStockInvalid(t *testing.T) {
	var e domain.StockEmplacement
	errs := applyStock(&e, StockEmplacementInput{Shift1: f(-1), Info: dailyinput.PtrStr("12345678901")})
	if len(errs) != 4 { // partner & tanggal wajib, info > 10, shift_1 < 0
		t.Errorf("kesalahan = %d, ingin 4: %v", len(errs), errs)
	}
}

func TestParseImportRow(t *testing.T) {
	s := &Service{}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	// Partner | Date | Awal | Scrapper | Crusher | Info | Shift 1 ...
	row := []string{"CPM", "01/10/2026", "100", "", "abc", "OK", "5"}
	key, rec, errs := s.ParseImportRow(pd, row)
	if key != "CPM|2026-10-01" {
		t.Errorf("kunci = %q", key)
	}
	if len(errs) != 1 || errs[0].Column != "Stock Cargo Stockpile Crusher" {
		t.Errorf("kesalahan = %v", errs)
	}
	if e := rec.(domain.StockEmplacement); *e.Shift1 != 5 || *e.Info != "OK" || e.ProjectID != pd.Project.ID {
		t.Errorf("kolom Info/Shift 1 tertukar atau project tidak terbawa: %+v", e)
	}
}

// ---------- stok per project (database palsu) ----------

type fakeRepo struct {
	rows map[int64]domain.StockEmplacement
	next int64
}

func (r *fakeRepo) List(_ context.Context, fl dailyinput.DailyFilter) ([]domain.StockEmplacement, error) {
	out := []domain.StockEmplacement{}
	for _, e := range r.rows {
		if fl.ProjectID == nil || e.ProjectID == *fl.ProjectID {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(_ context.Context, id int64) (*domain.StockEmplacement, error) {
	if e, ok := r.rows[id]; ok {
		return &e, nil
	}
	return nil, nil
}
func (r *fakeRepo) Upsert(_ context.Context, e domain.StockEmplacement, _ uuid.UUID) (int64, bool, error) {
	for id, cur := range r.rows {
		if cur.ProjectID == e.ProjectID && cur.Partner == e.Partner && cur.OperationalDate == e.OperationalDate {
			e.ID = id
			r.rows[id] = e
			return id, false, nil
		}
	}
	r.next++
	e.ID = r.next
	r.rows[e.ID] = e
	return e.ID, true, nil
}
func (r *fakeRepo) Update(_ context.Context, e domain.StockEmplacement, _ uuid.UUID) error {
	r.rows[e.ID] = e
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, id int64) error { delete(r.rows, id); return nil }
func (r *fakeRepo) BulkUpsert(context.Context, []domain.StockEmplacement, uuid.UUID, bool) (int, int, error) {
	return 0, 0, nil
}

type fakeProjects struct{}

func (fakeProjects) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return &projectdomain.Project{ID: id, Timezone: "Asia/Jakarta"}, nil
}

type noShifts struct{}

func (noShifts) ListActive(context.Context, uuid.UUID) ([]shiftdomain.Shift, error) { return nil, nil }

type noEquipment struct{}

func (noEquipment) CodeMap(context.Context, uuid.UUID) (map[string]uuid.UUID, error) { return nil, nil }

type noRefs struct{}

func (noRefs) List(context.Context, string) ([]referencedomain.ReferenceValue, error) {
	return nil, nil
}

func newService() *Service {
	loader := dailyinput.Loader{Projects: fakeProjects{}, Shifts: noShifts{}, Equipment: noEquipment{}, References: noRefs{}}
	return New(&fakeRepo{rows: map[int64]domain.StockEmplacement{}}, loader)
}

func actorOf(role string, pid uuid.UUID) access.Actor {
	return access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: role}
}

func input() StockEmplacementInput {
	return StockEmplacementInput{Partner: dailyinput.PtrStr("CPM"), OperationalDate: dailyinput.PtrStr("01/10/2026"), StockInitial: f(100)}
}

func TestStokMilikProjectPemanggilDanKunciPerProject(t *testing.T) {
	svc := newService()
	emp, chr := actorOf(access.RoleAdmin, uuid.New()), actorOf(access.RoleAdmin, uuid.New())
	a, created, err := svc.Set(context.Background(), emp, input())
	if err != nil || !created || a.ProjectID != *emp.ProjectID {
		t.Fatalf("set EMP: created=%v err=%v %+v", created, err, a)
	}
	// partner & tanggal yang sama pada project LAIN = baris baru (kunci menyertakan project)
	b, created, err := svc.Set(context.Background(), chr, input())
	if err != nil || !created || b.ID == a.ID || b.ProjectID != *chr.ProjectID {
		t.Errorf("project lain harus punya baris sendiri: created=%v err=%v", created, err)
	}
	// set ulang di project yang sama menimpa
	if _, created, _ := svc.Set(context.Background(), emp, input()); created {
		t.Error("kunci sama di project yang sama harus menimpa")
	}
}

func TestListDibatasiProjectDanLintasProjectDitolak(t *testing.T) {
	svc := newService()
	emp, chr := actorOf(access.RoleAdmin, uuid.New()), actorOf(access.RoleViewer, uuid.New())
	_, _, _ = svc.Set(context.Background(), emp, input())
	if list, _ := svc.List(context.Background(), chr, dailyinput.DailyFilter{}); len(list) != 0 {
		t.Errorf("project lain tidak boleh melihat stok EMP: %d baris", len(list))
	}
	if list, _ := svc.List(context.Background(), emp, dailyinput.DailyFilter{}); len(list) != 1 {
		t.Errorf("project sendiri harus melihat 1 baris: %d", len(list))
	}
	if _, err := svc.List(context.Background(), chr, dailyinput.DailyFilter{ProjectID: emp.ProjectID}); err == nil {
		t.Error("meminta project lain harus ditolak")
	}
}

func TestUbahDanHapusLintasProjectDitolakViewerTidakBolehMengisi(t *testing.T) {
	svc := newService()
	emp := actorOf(access.RoleAdmin, uuid.New())
	got, _, _ := svc.Set(context.Background(), emp, input())
	other := actorOf(access.RoleAdmin, uuid.New())
	if _, err := svc.Update(context.Background(), other, got.ID, StockEmplacementInput{Info: dailyinput.PtrStr("X")}); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("ubah lintas project harus 404: %v", err)
	}
	if err := svc.Delete(context.Background(), other, got.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("hapus lintas project harus 404: %v", err)
	}
	if _, _, err := svc.Set(context.Background(), actorOf(access.RoleViewer, *emp.ProjectID), input()); !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("viewer harus ditolak: %v", err)
	}
	if err := svc.Delete(context.Background(), emp, got.ID); err != nil {
		t.Errorf("admin project sendiri boleh menghapus: %v", err)
	}
}
