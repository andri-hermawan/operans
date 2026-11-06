package masterdata

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

var (
	projCHR = uuid.New()
	eqDT    = uuid.New()
	wib     = time.FixedZone("WIB", 7*3600)
)

type fakeStore struct {
	mu        sync.Mutex
	operator  string
	breakdown []BreakdownRow
	loads     map[string]int
	failOps   bool
}

func (f *fakeStore) count(p string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loads == nil {
		f.loads = map[string]int{}
	}
	f.loads[p]++
}

func (f *fakeStore) LoadShifts(context.Context) ([]ShiftRow, error) {
	f.count(PartProjects)
	return []ShiftRow{
		{ProjectID: projCHR, ProjectCode: "CHR", Timezone: "Asia/Jakarta", Code: "SHIFT_1", StartTime: "07:00", EndTime: "19:00", Sequence: 1},
		{ProjectID: projCHR, ProjectCode: "CHR", Timezone: "Asia/Jakarta", Code: "SHIFT_2", StartTime: "19:00", EndTime: "07:00", Sequence: 2},
	}, nil
}
func (f *fakeStore) LoadEquipment(context.Context) ([]EquipmentRow, error) {
	f.count(PartEquipment)
	return []EquipmentRow{{ID: eqDT, ProjectID: projCHR, Code: "dt10300"}}, nil
}
func (f *fakeStore) LoadOperators(context.Context) ([]OperatorRow, error) {
	f.count(PartOperators)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOps {
		return nil, errors.New("db mati")
	}
	return []OperatorRow{{ProjectID: projCHR, OperationalDate: "2026-10-22", ShiftCode: "SHIFT_2", EquipmentCode: "DT10300", OperatorName: f.operator}}, nil
}
func (f *fakeStore) LoadBreakdowns(context.Context) ([]BreakdownRow, error) {
	f.count(PartBreakdowns)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.breakdown, nil
}

func at(day, h, m int) time.Time { return time.Date(2026, 10, day, h, m, 0, 0, wib) }
func ptr[T any](v T) *T          { return &v }

func event(t time.Time) *domain.Event { return &domain.Event{EventTime: t.UTC()} }

func TestEnrich(t *testing.T) {
	// satu catatan breakdown (jam 23:00–01:30). Status MENEMPEL: tetap breakdown di luar jam itu dan di shift lain,
	// sampai ada status ready yang diinput.
	st := &fakeStore{operator: "Budi", breakdown: []BreakdownRow{
		{ProjectID: projCHR, OperationalDate: "2026-10-22", ShiftCode: "SHIFT_2", EquipmentCode: "DT10300",
			Status: "breakdown", StartedAt: ptr(at(22, 23, 0)), EndedAt: ptr(at(23, 1, 30)), Description: ptr("AC HOT"), UpdatedAt: at(22, 23, 5)},
	}}
	c := New(st)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		t        time.Time
		operator string // kosong = tidak ada operator pada shift itu
	}{
		{"sebelum jam mulai breakdown (22:00)", at(22, 22, 0), "Budi"},
		{"saat breakdown (00:30, lewat tengah malam)", at(23, 0, 30), "Budi"},
		{"lewat jam selesai (02:00)", at(23, 2, 0), "Budi"},
		{"shift 1 hari berikutnya", at(23, 10, 0), ""},
	}
	for _, cs := range cases {
		e := event(cs.t)
		c.Enrich(&projCHR, &eqDT, e)
		if e.OperationalDate == nil || e.ShiftCode == nil {
			t.Errorf("%s: periode tidak terisi", cs.name)
			continue
		}
		if e.IsBreakdown == nil || !*e.IsBreakdown || e.BreakdownDescription == nil || *e.BreakdownDescription != "AC HOT" {
			t.Errorf("%s: breakdown=%v desc=%v (harus tetap breakdown, deskripsi AC HOT)", cs.name, e.IsBreakdown, e.BreakdownDescription)
		}
		if (cs.operator == "") != (e.OperatorName == nil) || (e.OperatorName != nil && *e.OperatorName != cs.operator) {
			t.Errorf("%s: operator=%v", cs.name, e.OperatorName)
		}
	}
	if e := event(at(22, 23, 30)); true {
		c.Enrich(&projCHR, &eqDT, e)
		if *e.OperationalDate != "2026-10-22" || *e.ShiftCode != "SHIFT_2" {
			t.Errorf("periode: %v %v", e.OperationalDate, e.ShiftCode)
		}
	}

	// status ready diinput (lebih baru): breakdown berhenti & deskripsi hilang
	st.mu.Lock()
	st.breakdown = append(st.breakdown, BreakdownRow{ProjectID: projCHR, OperationalDate: "2026-10-23", ShiftCode: "SHIFT_1", EquipmentCode: "DT10300",
		Status: "ready", Description: ptr("selesai"), UpdatedAt: at(23, 9, 0)})
	st.mu.Unlock()
	if err := c.Reload(context.Background(), PartBreakdowns); err != nil {
		t.Fatal(err)
	}
	e := event(at(23, 10, 0))
	c.Enrich(&projCHR, &eqDT, e)
	if e.IsBreakdown == nil || *e.IsBreakdown || e.BreakdownDescription != nil {
		t.Errorf("setelah ready: breakdown=%v desc=%v", e.IsBreakdown, e.BreakdownDescription)
	}

	// unit tanpa catatan breakdown: dibiarkan kosong (service ingest yang mengisi nilai yang sedang berlaku / false)
	st.mu.Lock()
	st.breakdown = nil
	st.mu.Unlock()
	_ = c.Reload(context.Background(), PartBreakdowns)
	e = event(at(22, 10, 0))
	c.Enrich(&projCHR, &eqDT, e)
	if *e.ShiftCode != "SHIFT_1" || e.OperatorName != nil || e.IsBreakdown != nil {
		t.Errorf("tanpa catatan: %+v", e.Derived)
	}
	// project / equipment tidak dikenal → tidak diisi apa pun
	e = event(at(22, 10, 0))
	other := uuid.New()
	c.Enrich(&other, &eqDT, e)
	if e.ShiftCode != nil {
		t.Error("project tak dikenal seharusnya tidak diisi")
	}
	e = event(at(22, 10, 0))
	c.Enrich(&projCHR, &other, e)
	if e.ShiftCode == nil || e.IsBreakdown != nil {
		t.Error("equipment tak dikenal: shift diisi, operator/breakdown kosong")
	}
}

// fakeNotifier mengirim payload dari channel; tanpa pesan → menunggu sampai ctx berakhir.
type fakeNotifier struct{ ch chan string }

func (n *fakeNotifier) Next(ctx context.Context) (string, error) {
	select {
	case p := <-n.ch:
		if p == "PUTUS" {
			return "", errors.New("koneksi putus")
		}
		return p, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestRunNotify(t *testing.T) {
	st := &fakeStore{operator: "Budi"}
	c := New(st)
	_ = c.Load(context.Background())
	n := &fakeNotifier{ch: make(chan string, 10)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.Run(ctx, n, time.Hour, 50*time.Millisecond); close(done) }()

	operatorNow := func() string {
		e := event(at(23, 2, 0))
		c.Enrich(&projCHR, &eqDT, e)
		if e.OperatorName == nil {
			return ""
		}
		return *e.OperatorName
	}
	waitFor := func(want string) {
		t.Helper()
		for i := 0; i < 100 && operatorNow() != want; i++ {
			time.Sleep(10 * time.Millisecond)
		}
		if got := operatorNow(); got != want {
			t.Fatalf("operator = %q, mau %q", got, want)
		}
	}

	// banyak NOTIFY beruntun (mis. impor Excel) → satu kali muat ulang bagian operator saja
	st.mu.Lock()
	st.operator = "Andi"
	st.mu.Unlock()
	before := st.loads[PartOperators]
	for i := 0; i < 5; i++ {
		n.ch <- "core.operator_assignments"
	}
	n.ch <- "core.alert_rules" // tabel yang belum dipakai → diabaikan
	waitFor("Andi")
	time.Sleep(100 * time.Millisecond)
	st.mu.Lock()
	if got := st.loads[PartOperators] - before; got != 1 || st.loads[PartProjects] != 1 {
		t.Errorf("muat ulang operator %d kali (mau 1), projects %d (mau 1 dari Load awal)", got, st.loads[PartProjects])
	}
	// muat ulang gagal → data lama tetap dipakai
	st.failOps = true
	st.mu.Unlock()
	n.ch <- "core.operator_assignments"
	time.Sleep(150 * time.Millisecond)
	waitFor("Andi")

	// koneksi putus → setelah jeda, semua bagian dimuat ulang
	st.mu.Lock()
	st.failOps = false
	st.operator = "Citra"
	st.mu.Unlock()
	n.ch <- "PUTUS"
	time.Sleep(1200 * time.Millisecond) // jeda sambung ulang pertama = 1 detik
	waitFor("Citra")

	cancel()
	<-done
}

func (f *fakeStore) LoadZones(context.Context) ([]ZoneRow, error) {
	f.count(PartZones)
	return nil, nil
}

func (f *fakeStore) LoadCalibrations(context.Context) ([]CalibrationRow, error) {
	f.count(PartCalibrations)
	return nil, nil
}

func (f *fakeStore) LoadAlertRules(context.Context) ([]AlertRuleRow, error) {
	f.count(PartAlertRules)
	return nil, nil
}

func (f *fakeStore) LoadRecipients(context.Context) ([]Recipient, error) {
	f.count(PartRecipients)
	return nil, nil
}
