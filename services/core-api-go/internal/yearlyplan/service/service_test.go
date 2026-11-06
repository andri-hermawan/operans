package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/plan"
	shiftdomain "simha/core-api/internal/shift/domain"
	"simha/core-api/internal/yearlyplan/domain"
)

func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }
func s(v string) *string   { return &v }

// Contoh baris dari tabel lama (Plan Yearly TBBE 2025, PIT A2).
var sampleMonths = []float64{180055.41, 132485.47, 45757.93, 176691.46, 213976.63, 147409.45, 104064.95, 102978.01, 111148.41, 83034.93, 27895.43, 0}

func sampleInput() YearlyPlanInput {
	return YearlyPlanInput{Partner: s("TBBE"), PlanYear: i(2025), Location: s("PIT A2"), Description: s("Volume"),
		PlanName: s("Waste Removal"), Months: plan.Pointers(sampleMonths)}
}

func TestApplyYearlyTotalKosongDihitungDariBulan(t *testing.T) {
	var p domain.YearlyPlan
	if errs := applyYearly(&p, sampleInput()); len(errs) != 0 {
		t.Fatalf("seharusnya valid: %v", errs)
	}
	if p.Total != 1325498.08 || p.Months[0] != 180055.41 || p.Months[11] != 0 || p.Partner != "TBBE" || p.PlanYear != 2025 {
		t.Errorf("hasil salah: %+v", p)
	}
}

func TestApplyYearlyTotalDiisiDisimpanApaAdanya(t *testing.T) {
	var p domain.YearlyPlan
	in := sampleInput()
	in.Total = f(1325498.10) // beda tipis dari jumlah: data lama memang bisa begitu (pembulatan)
	if errs := applyYearly(&p, in); len(errs) != 0 {
		t.Fatalf("total yang diisi tidak boleh ditolak: %v", errs)
	}
	if p.Total != 1325498.10 {
		t.Errorf("total = %v, ingin 1325498.10", p.Total)
	}
}

func TestApplyYearlyKosongDianggapNolDanDibulatkan(t *testing.T) {
	var p domain.YearlyPlan
	in := sampleInput()
	in.Months = make([]*float64, 12)
	in.Months[0], in.Months[1] = f(10.456), nil
	if errs := applyYearly(&p, in); len(errs) != 0 {
		t.Fatal(errs)
	}
	if p.Months[0] != 10.46 || p.Months[1] != 0 || p.Total != 10.46 {
		t.Errorf("hasil salah: %+v", p)
	}
}

func TestApplyYearlyKesalahan(t *testing.T) {
	var p domain.YearlyPlan
	in := YearlyPlanInput{PlanYear: i(1999), Months: make([]*float64, 12)}
	in.Months[2] = f(-1)
	errs := applyYearly(&p, in)
	// partner, location, description, plan_name wajib (4) + tahun di luar rentang + mar negatif
	if len(errs) != 6 {
		t.Errorf("kesalahan = %d, ingin 6: %v", len(errs), errs)
	}
}

func TestMarshalYearlyDatar(t *testing.T) {
	var p domain.YearlyPlan
	applyYearly(&p, sampleInput())
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["jan"] != 180055.41 || m["dec"] != 0.0 || m["total"] != 1325498.08 || m["plan_name"] != "Waste Removal" {
		t.Errorf("JSON tidak datar/benar: %s", b)
	}
	if _, ada := m["months"]; ada {
		t.Error("bulan tidak boleh berupa larik")
	}
}

func TestUnmarshalInputDatar(t *testing.T) {
	var in YearlyPlanInput
	body := `{"partner":"TBBE","plan_year":2025,"location":"PIT A2","description":"Volume","plan_name":"Waste Removal","jan":10.5,"dec":null,"jun":3}`
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		t.Fatal(err)
	}
	if *in.Partner != "TBBE" || *in.PlanYear != 2025 || *in.Months[0] != 10.5 || in.Months[11] != nil || *in.Months[5] != 3 || in.Months[1] != nil {
		t.Errorf("hasil salah: %+v", in)
	}
}

func TestUnmarshalInputMenolakIsianTakDikenalDanBukanAngka(t *testing.T) {
	for _, body := range []string{`{"partner":"X","bulan_ke_13":1}`, `{"jan":"abc"}`, `{"plan_year":"dua ribu"}`} {
		var in YearlyPlanInput
		if err := json.Unmarshal([]byte(body), &in); err == nil {
			t.Errorf("seharusnya ditolak: %s", body)
		}
	}
}

func TestParseImportRowContohLama(t *testing.T) {
	row := []string{"TBBE", "2025", "PIT A2", "Volume", "Waste Removal", "1325498.08",
		"180055.41", "132485.47", "45757.93", "176691.46", "213976.63", "147409.45", "104064.95", "102978.01", "111148.41", "83034.93", "27895.43", "0.00"}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	key, rec, errs := (&Service{}).ParseImportRow(pd, row)
	if len(errs) != 0 {
		t.Fatalf("seharusnya valid: %v", errs)
	}
	if key != "TBBE|2025|PIT A2|Volume|Waste Removal" {
		t.Errorf("kunci = %q", key)
	}
	if p := rec.(domain.YearlyPlan); p.ProjectID != pd.Project.ID || p.Total != 1325498.08 || p.Months[4] != 213976.63 {
		t.Errorf("rekaman salah: %+v", p)
	}
}

func TestParseImportRowKesalahanMemakaiJudulKolomTemplate(t *testing.T) {
	row := []string{"TBBE", "2025.5", "PIT A2", "Volume", "Waste Removal", "", "abc"}
	pd := &dailyinput.ProjectData{Project: &projectdomain.Project{ID: uuid.New()}}
	_, _, errs := (&Service{}).ParseImportRow(pd, row)
	cols := map[string]bool{}
	for _, e := range errs {
		cols[e.Column] = true
	}
	if !cols["Year"] || !cols["Jan"] {
		t.Errorf("kesalahan harus menyebut kolom Year dan Jan: %v", errs)
	}
}

func TestImportColumnsSelaras(t *testing.T) {
	headers, example := (&Service{}).ImportColumns()
	if len(headers) != 18 || len(example) != 18 || headers[0] != "Mitra" || headers[17] != "Dec" {
		t.Errorf("template salah: %v (%d contoh)", headers, len(example))
	}
}

// ---------- alur service dengan database palsu ----------

type fakeRepo struct {
	rows map[int64]domain.YearlyPlan
	next int64
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[int64]domain.YearlyPlan{}} }

func (r *fakeRepo) List(_ context.Context, fl plan.Filter) ([]domain.YearlyPlan, error) {
	out := []domain.YearlyPlan{}
	for _, p := range r.rows {
		if (fl.ProjectID == nil || p.ProjectID == *fl.ProjectID) && (fl.Year == 0 || p.PlanYear == fl.Year) {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *fakeRepo) Get(_ context.Context, id int64) (*domain.YearlyPlan, error) {
	if p, ok := r.rows[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *fakeRepo) Upsert(_ context.Context, p domain.YearlyPlan, _ uuid.UUID) (int64, bool, error) {
	for id, cur := range r.rows {
		if cur.ProjectID == p.ProjectID && cur.Partner == p.Partner && cur.PlanYear == p.PlanYear && cur.Location == p.Location &&
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
func (r *fakeRepo) Update(_ context.Context, p domain.YearlyPlan, _ uuid.UUID) error {
	r.rows[p.ID] = p
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, id int64) error { delete(r.rows, id); return nil }
func (r *fakeRepo) BulkUpsert(context.Context, []domain.YearlyPlan, uuid.UUID, bool) (int, int, error) {
	return 0, 0, nil
}

type fakeProjects struct{ id uuid.UUID }

func (p fakeProjects) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
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
	pid := uuid.New()
	repo := newFakeRepo()
	loader := dailyinput.Loader{Projects: fakeProjects{pid}, Shifts: noShifts{}, Equipment: noEquipment{}, References: noRefs{}}
	return New(repo, loader), repo, pid
}

func actorOf(role string, pid uuid.UUID) access.Actor {
	return access.Actor{UserID: uuid.New(), ProjectID: &pid, Role: role}
}

func TestSetLaluSetUlangMenimpaBarisYangSama(t *testing.T) {
	svc, repo, pid := newService()
	a := actorOf(access.RoleDispatcher, pid)
	_, created, err := svc.Set(context.Background(), a, sampleInput())
	if err != nil || !created {
		t.Fatalf("set pertama: created=%v err=%v", created, err)
	}
	in := sampleInput()
	in.Months = plan.Pointers(append([]float64{1}, sampleMonths[1:]...))
	got, created, err := svc.Set(context.Background(), a, in)
	if err != nil || created || len(repo.rows) != 1 || got.Months[0] != 1 {
		t.Errorf("set kedua harus menimpa: created=%v rows=%d err=%v", created, len(repo.rows), err)
	}
}

func TestViewerTidakBolehMengisi(t *testing.T) {
	svc, _, pid := newService()
	_, _, err := svc.Set(context.Background(), actorOf(access.RoleViewer, pid), sampleInput())
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("viewer harus ditolak, err = %v", err)
	}
}

func TestUpdateMengubahBulanMenghitungUlangTotalKecualiTotalDiberikan(t *testing.T) {
	svc, _, pid := newService()
	a := actorOf(access.RoleAdmin, pid)
	got, _, _ := svc.Set(context.Background(), a, sampleInput())

	var patch YearlyPlanInput
	_ = json.Unmarshal([]byte(`{"jan":100000}`), &patch)
	upd, err := svc.Update(context.Background(), a, got.ID, patch)
	if err != nil {
		t.Fatal(err)
	}
	if upd.Months[0] != 100000 || upd.Total != plan.Sum(upd.Months[:]) { // total ikut dihitung ulang
		t.Errorf("total tidak dihitung ulang: %+v", upd)
	}

	patch = YearlyPlanInput{}
	_ = json.Unmarshal([]byte(`{"jan":5,"total":999}`), &patch)
	upd, _ = svc.Update(context.Background(), a, got.ID, patch)
	if upd.Total != 999 { // total yang diberikan dipakai apa adanya
		t.Errorf("total = %v, ingin 999", upd.Total)
	}

	patch = YearlyPlanInput{}
	_ = json.Unmarshal([]byte(`{"location":"PIT B1"}`), &patch)
	upd, _ = svc.Update(context.Background(), a, got.ID, patch)
	if upd.Location != "PIT B1" || upd.Total != 999 || upd.Months[0] != 5 { // ubah teks saja: angka & total tetap
		t.Errorf("ubah lokasi tidak boleh mengubah angka: %+v", upd)
	}
}

func TestProjectLainTidakBisaMengubahDanMenghapus(t *testing.T) {
	svc, _, pid := newService()
	got, _, _ := svc.Set(context.Background(), actorOf(access.RoleAdmin, pid), sampleInput())
	orang := actorOf(access.RoleAdmin, uuid.New()) // admin project lain
	if _, err := svc.Update(context.Background(), orang, got.ID, YearlyPlanInput{Partner: s("X")}); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("ubah lintas project harus 404, err = %v", err)
	}
	if err := svc.Delete(context.Background(), orang, got.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("hapus lintas project harus 404, err = %v", err)
	}
	if err := svc.Delete(context.Background(), actorOf(access.RoleViewer, pid), got.ID); !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("viewer tidak boleh menghapus, err = %v", err)
	}
	if err := svc.Delete(context.Background(), actorOf(access.RoleAdmin, pid), got.ID); err != nil {
		t.Errorf("admin boleh menghapus: %v", err)
	}
}

func TestListMembatasiProjectSesuaiHak(t *testing.T) {
	svc, _, pid := newService()
	_, _, _ = svc.Set(context.Background(), actorOf(access.RoleAdmin, pid), sampleInput())
	list, err := svc.List(context.Background(), actorOf(access.RoleViewer, pid), plan.Filter{Year: 2025})
	if err != nil || len(list) != 1 {
		t.Errorf("viewer harus melihat 1 baris: %v %v", len(list), err)
	}
	other := uuid.New()
	if _, err := svc.List(context.Background(), actorOf(access.RoleViewer, pid), plan.Filter{ProjectID: &other}); err == nil {
		t.Error("melihat project lain harus ditolak")
	}
	if _, err := svc.List(context.Background(), actorOf(access.RoleViewer, pid), plan.Filter{Year: 1900}); err == nil || !strings.Contains(err.Error(), "year") {
		t.Errorf("tahun tidak masuk akal harus ditolak: %v", err)
	}
}
