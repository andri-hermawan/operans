package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"simha/core-api/internal/monthlyplan/domain"
	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/plan"
	shiftdomain "simha/core-api/internal/shift/domain"
)

func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }
func s(v string) *string   { return &v }

// Contoh baris dari tabel lama (Plan Month TBBE, Juni 2025, Waste Removal, BCM): T01..T31.
var sampleDays = []float64{5075.38, 5075.38, 5075.38, 5075.38, 5075.38, 4485.98, 4972.25, 4972.25, 4972.25, 4917.19, 4972.25, 4972.25, 4394.82,
	4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4394.82, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4972.25, 4394.82,
	4972.25, 4972.25, 4972.25, 0}

func sampleInput() MonthlyPlanInput {
	return MonthlyPlanInput{Partner: s("TBBE"), PlanYear: i(2025), PlanMonth: i(6), Description: s("Volume"), PlanName: s("Waste Removal"),
		Uom: s("BCM"), Total: f(147409.45), Days: plan.Pointers(sampleDays)}
}

func TestApplyMonthlyContohLama(t *testing.T) {
	var p domain.MonthlyPlan
	if errs := applyMonthly(&p, sampleInput()); len(errs) != 0 {
		t.Fatalf("contoh dari tabel lama harus valid: %v", errs)
	}
	// total pada data lama (147409.45) sedikit beda dari jumlah tanggal (147409.53): disimpan apa adanya, tidak ditolak
	if p.Total != 147409.45 || plan.Sum(p.Days[:]) != 147409.53 || p.Days[30] != 0 || p.Uom != "BCM" {
		t.Errorf("hasil salah: total=%v jumlah=%v", p.Total, plan.Sum(p.Days[:]))
	}
}

func TestApplyMonthlyTotalKosongDihitungDariTanggal(t *testing.T) {
	var p domain.MonthlyPlan
	in := sampleInput()
	in.Total = nil
	if errs := applyMonthly(&p, in); len(errs) != 0 {
		t.Fatal(errs)
	}
	if p.Total != 147409.53 {
		t.Errorf("total = %v, ingin 147409.53", p.Total)
	}
}

func TestTanggalDiLuarPanjangBulanHarusNol(t *testing.T) {
	cases := []struct {
		year, month, day int // day = nomor tanggal (1..31) yang diisi 100
		ok               bool
	}{
		{2025, 6, 30, true},  // Juni 30 hari
		{2025, 6, 31, false}, // Juni tidak punya tanggal 31
		{2024, 2, 29, true},  // 2024 kabisat
		{2025, 2, 29, false}, // 2025 bukan kabisat
		{2025, 2, 30, false},
		{2025, 1, 31, true},
		{2025, 4, 31, false},
	}
	for _, c := range cases {
		in := sampleInput()
		in.Total = nil
		in.PlanYear, in.PlanMonth = i(c.year), i(c.month)
		in.Days = make([]*float64, 31)
		in.Days[c.day-1] = f(100)
		var p domain.MonthlyPlan
		errs := applyMonthly(&p, in)
		if (len(errs) == 0) != c.ok {
			t.Errorf("%d/%d tanggal %d: ok=%v, kesalahan=%v", c.month, c.year, c.day, c.ok, errs)
		}
	}
}

func TestApplyMonthlyKesalahan(t *testing.T) {
	var p domain.MonthlyPlan
	errs := applyMonthly(&p, MonthlyPlanInput{PlanYear: i(2025), PlanMonth: i(13), Days: make([]*float64, 31)})
	// partner, description, plan_name, uom wajib (4) + bulan 13
	if len(errs) != 5 {
		t.Errorf("kesalahan = %d, ingin 5: %v", len(errs), errs)
	}
}

func TestMarshalMonthlyDatar(t *testing.T) {
	var p domain.MonthlyPlan
	applyMonthly(&p, sampleInput())
	b, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["day_01"] != 5075.38 || m["day_31"] != 0.0 || m["plan_month"] != 6.0 || m["uom"] != "BCM" || m["total"] != 147409.45 {
		t.Errorf("JSON tidak datar/benar: %s", b)
	}
	if _, ada := m["days"]; ada {
		t.Error("tanggal tidak boleh berupa larik")
	}
}

func TestUnmarshalMonthlyMenolakIsianTakDikenal(t *testing.T) {
	for _, body := range []string{`{"day_32":1}`, `{"T01":1}`, `{"day_01":"x"}`} {
		var in MonthlyPlanInput
		if err := json.Unmarshal([]byte(body), &in); err == nil {
			t.Errorf("seharusnya ditolak: %s", body)
		}
	}
	var in MonthlyPlanInput
	if err := json.Unmarshal([]byte(`{"day_01":1.5,"day_31":null}`), &in); err != nil || *in.Days[0] != 1.5 || in.Days[30] != nil {
		t.Errorf("isian valid ditolak/salah: %v %+v", err, in)
	}
}

func TestParseImportRowContohLama(t *testing.T) {
	row := []string{"TBBE", "2025", "6", "Volume", "Waste Removal", "BCM", "147409.45"}
	for _, d := range sampleDays {
		row = append(row, strings.TrimRight(strings.TrimRight(jsonNum(d), "0"), "."))
	}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	key, rec, errs := (&Service{}).ParseImportRow(pd, row)
	if len(errs) != 0 {
		t.Fatalf("seharusnya valid: %v", errs)
	}
	if key != "TBBE|2025|6|Volume|Waste Removal" {
		t.Errorf("kunci = %q", key)
	}
	if p := rec.(domain.MonthlyPlan); p.Total != 147409.45 || p.Days[9] != 4917.19 || p.Days[30] != 0 {
		t.Errorf("rekaman salah: %+v", p)
	}
}

func jsonNum(v float64) string { b, _ := json.Marshal(v); return string(b) }

func TestParseImportRowTanggal31DiJuniDitolakDenganJudulKolomTemplate(t *testing.T) {
	row := []string{"TBBE", "2025", "6", "Volume", "Waste Removal", "BCM", ""}
	for d := 1; d <= 31; d++ {
		if d == 31 {
			row = append(row, "10")
		} else {
			row = append(row, "0")
		}
	}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	_, _, errs := (&Service{}).ParseImportRow(pd, row)
	if len(errs) != 1 || errs[0].Column != "T31" {
		t.Errorf("harus 1 kesalahan di kolom T31: %v", errs)
	}
}

func TestImportColumnsSelaras(t *testing.T) {
	headers, example := (&Service{}).ImportColumns()
	if len(headers) != 38 || len(example) != 38 || headers[0] != "Mitra" || headers[5] != "Uom" || headers[7] != "T01" || headers[37] != "T31" {
		t.Errorf("template salah: %v (%d contoh)", headers, len(example))
	}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	row := make([]string, len(example))
	for k, v := range example {
		row[k] = jsonString(v)
	}
	if _, _, errs := (&Service{}).ParseImportRow(pd, row); len(errs) != 0 {
		t.Errorf("baris contoh template harus lolos impor: %v", errs)
	}
}

func jsonString(v any) string { b, _ := json.Marshal(v); return strings.Trim(string(b), `"`) }

// ---------- alur service dengan database palsu ----------

type fakeRepo struct {
	rows map[int64]domain.MonthlyPlan
	next int64
}

func (r *fakeRepo) List(_ context.Context, fl plan.Filter) ([]domain.MonthlyPlan, error) {
	out := []domain.MonthlyPlan{}
	for _, p := range r.rows {
		if (fl.ProjectID == nil || p.ProjectID == *fl.ProjectID) && (fl.Year == 0 || p.PlanYear == fl.Year) && (fl.Month == 0 || p.PlanMonth == fl.Month) {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(_ context.Context, id int64) (*domain.MonthlyPlan, error) {
	if p, ok := r.rows[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *fakeRepo) Upsert(_ context.Context, p domain.MonthlyPlan, _ uuid.UUID) (int64, bool, error) {
	for id, cur := range r.rows {
		if cur.ProjectID == p.ProjectID && cur.Partner == p.Partner && cur.PlanYear == p.PlanYear && cur.PlanMonth == p.PlanMonth &&
			cur.Description == p.Description && cur.PlanName == p.PlanName {
			p.ID = id
			r.rows[id] = p
			return id, false, nil
		}
	}
	r.next++
	p.ID = r.next
	r.rows[p.ID] = p
	return p.ID, true, nil
}
func (r *fakeRepo) Update(_ context.Context, p domain.MonthlyPlan, _ uuid.UUID) error {
	r.rows[p.ID] = p
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, id int64) error { delete(r.rows, id); return nil }
func (r *fakeRepo) BulkUpsert(context.Context, []domain.MonthlyPlan, uuid.UUID, bool) (int, int, error) {
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

func newService() (*Service, *fakeRepo, uuid.UUID) {
	repo := &fakeRepo{rows: map[int64]domain.MonthlyPlan{}}
	loader := dailyinput.Loader{Projects: fakeProjects{}, Shifts: noShifts{}, Equipment: noEquipment{}, References: noRefs{}}
	return New(repo, loader), repo, uuid.New()
}

func actorOf(role string, pid uuid.UUID) access.Actor {
	return access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: role}
}

func TestSetMenimpaBarisYangSamaDanViewerDitolak(t *testing.T) {
	svc, repo, pid := newService()
	a := actorOf(access.RoleDispatcher, pid)
	if _, created, err := svc.Set(context.Background(), a, sampleInput()); err != nil || !created {
		t.Fatalf("set pertama: created=%v err=%v", created, err)
	}
	in := sampleInput()
	in.Uom = s("ton")
	if got, created, err := svc.Set(context.Background(), a, in); err != nil || created || len(repo.rows) != 1 || got.Uom != "ton" {
		t.Errorf("set kedua harus menimpa: created=%v rows=%d err=%v", created, len(repo.rows), err)
	}
	if _, _, err := svc.Set(context.Background(), actorOf(access.RoleViewer, pid), sampleInput()); !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("viewer harus ditolak, err = %v", err)
	}
}

func TestUpdateGantiBulanMemeriksaUlangTanggal(t *testing.T) {
	svc, _, pid := newService()
	a := actorOf(access.RoleAdmin, pid)
	got, _, _ := svc.Set(context.Background(), a, sampleInput()) // Juni: day_30 terisi
	var patch MonthlyPlanInput
	_ = json.Unmarshal([]byte(`{"plan_month":2}`), &patch) // Februari tidak punya tanggal 30
	if _, err := svc.Update(context.Background(), a, got.ID, patch); err == nil {
		t.Error("pindah ke Februari dengan tanggal 30 terisi harus ditolak")
	}
}

func TestUpdateTanggalMenghitungUlangTotalKecualiTotalDiberikan(t *testing.T) {
	svc, _, pid := newService()
	a := actorOf(access.RoleAdmin, pid)
	got, _, _ := svc.Set(context.Background(), a, sampleInput())
	var patch MonthlyPlanInput
	_ = json.Unmarshal([]byte(`{"day_01":1}`), &patch)
	upd, err := svc.Update(context.Background(), a, got.ID, patch)
	if err != nil || upd.Total != plan.Sum(upd.Days[:]) {
		t.Errorf("total harus dihitung ulang: %v %+v", err, upd.Total)
	}
	patch = MonthlyPlanInput{}
	_ = json.Unmarshal([]byte(`{"day_02":2,"total":777}`), &patch)
	if upd, _ = svc.Update(context.Background(), a, got.ID, patch); upd.Total != 777 {
		t.Errorf("total yang diberikan harus dipakai: %v", upd.Total)
	}
	patch = MonthlyPlanInput{}
	_ = json.Unmarshal([]byte(`{"uom":"ton"}`), &patch)
	if upd, _ = svc.Update(context.Background(), a, got.ID, patch); upd.Uom != "ton" || upd.Total != 777 || upd.Days[1] != 2 {
		t.Errorf("ubah uom tidak boleh mengubah angka: %+v", upd)
	}
}

func TestProjectLainTidakBisaMengubahDanMenghapusDanListDibatasi(t *testing.T) {
	svc, _, pid := newService()
	got, _, _ := svc.Set(context.Background(), actorOf(access.RoleAdmin, pid), sampleInput())
	orang := actorOf(access.RoleAdmin, uuid.New())
	if _, err := svc.Update(context.Background(), orang, got.ID, MonthlyPlanInput{Partner: s("X")}); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("ubah lintas project harus 404, err = %v", err)
	}
	if err := svc.Delete(context.Background(), orang, got.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("hapus lintas project harus 404, err = %v", err)
	}
	list, err := svc.List(context.Background(), actorOf(access.RoleViewer, pid), plan.Filter{Year: 2025, Month: 6})
	if err != nil || len(list) != 1 {
		t.Errorf("viewer harus melihat 1 baris: %v %v", len(list), err)
	}
	if _, err := svc.List(context.Background(), actorOf(access.RoleViewer, pid), plan.Filter{Month: 13}); err == nil {
		t.Error("bulan 13 harus ditolak")
	}
	if err := svc.Delete(context.Background(), actorOf(access.RoleAdmin, pid), got.ID); err != nil {
		t.Errorf("admin boleh menghapus: %v", err)
	}
}
