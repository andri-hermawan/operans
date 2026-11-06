package masterdata

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func row(status, date, shift string, upd int, desc *string) BreakdownRow {
	return BreakdownRow{ProjectID: projCHR, OperationalDate: date, ShiftCode: shift, EquipmentCode: "DT10300", Status: status,
		Description: desc, UpdatedAt: at(24, 0, 0).Add(time.Duration(upd) * time.Minute)}
}

// Status breakdown MENEMPEL: keadaan unit mengikuti catatan yang TERAKHIR diinput, tidak bergantung jam/shift/hari.
func TestStickyBreakdown(t *testing.T) {
	d := func(s string) *string { return &s }
	cases := []struct {
		name string
		rows []BreakdownRow // terbaru di depan
		is   bool
		desc string // "" = nil
		ok   bool
	}{
		{"belum ada catatan", nil, false, "", false},
		{"breakdown: tampil dengan deskripsinya", []BreakdownRow{row("breakdown", "2026-10-24", "SHIFT_1", 5, d("AC HOT"))}, true, "AC HOT", true},
		{"continue tanpa deskripsi: pakai deskripsi breakdown sebelumnya", []BreakdownRow{row("continue", "2026-10-24", "SHIFT_2", 9, nil), row("breakdown", "2026-10-24", "SHIFT_1", 5, d("AC HOT"))}, true, "AC HOT", true},
		{"continue dengan deskripsi baru", []BreakdownRow{row("continue", "2026-10-24", "SHIFT_2", 9, d("AC HOT, tunggu part")), row("breakdown", "2026-10-24", "SHIFT_1", 5, d("AC HOT"))}, true, "AC HOT, tunggu part", true},
		{"ready terbaru: normal & deskripsi hilang", []BreakdownRow{row("ready", "2026-10-25", "SHIFT_1", 20, d("selesai")), row("breakdown", "2026-10-24", "SHIFT_1", 5, d("AC HOT"))}, false, "", true},
		{"breakdown baru sesudah ready: deskripsi lama tidak ikut", []BreakdownRow{row("breakdown", "2026-10-26", "SHIFT_1", 30, nil), row("ready", "2026-10-25", "SHIFT_1", 20, d("selesai")), row("breakdown", "2026-10-24", "SHIFT_1", 5, d("AC HOT"))}, true, "", true},
	}
	for _, cs := range cases {
		is, desc, ok := stickyBreakdown(cs.rows)
		if is != cs.is || ok != cs.ok || (desc == nil) != (cs.desc == "") || (desc != nil && *desc != cs.desc) {
			t.Errorf("%s: is=%v desc=%v ok=%v", cs.name, is, desc, ok)
		}
	}
}

// Urutan "terbaru": waktu input terakhir; bila sama (satu unggahan) → tanggal, shift, jam mulai.
func TestBreakdownsByUnitOrder(t *testing.T) {
	same := at(24, 0, 0)
	mk := func(date, shift, status string, startH int) BreakdownRow {
		r := BreakdownRow{ProjectID: projCHR, OperationalDate: date, ShiftCode: shift, EquipmentCode: "DT10300", Status: status, UpdatedAt: same}
		if startH >= 0 {
			r.StartedAt = ptr(at(24, startH, 0))
		}
		return r
	}
	byDay := map[dayKey][]BreakdownRow{}
	for _, r := range []BreakdownRow{mk("2026-10-24", "SHIFT_1", "breakdown", 7), mk("2026-10-24", "SHIFT_1", "ready", 11), mk("2026-10-23", "SHIFT_2", "breakdown", 20)} {
		k := dayKey{r.ProjectID, r.OperationalDate, r.ShiftCode, r.EquipmentCode}
		byDay[k] = append(byDay[k], r)
	}
	got := breakdownsByUnit(byDay)[unitKey{projCHR, "DT10300"}]
	if len(got) != 3 || got[0].Status != "ready" || got[1].Status != "breakdown" || got[2].OperationalDate != "2026-10-23" {
		t.Fatalf("urutan: %+v", got)
	}
}

// Keadaan unit dari data induk: breakdown menempel + operator menurut shift.
func TestOperationalAt(t *testing.T) {
	st := &fakeStore{operator: "Dede", breakdown: []BreakdownRow{
		{ProjectID: projCHR, OperationalDate: "2026-10-22", ShiftCode: "SHIFT_2", EquipmentCode: "DT10300",
			Status: "breakdown", StartedAt: ptr(at(22, 23, 0)), EndedAt: ptr(at(23, 1, 30)), Description: ptr("AC HOT"), UpdatedAt: at(22, 23, 5)},
	}}
	c := New(st)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	// sehari kemudian, jauh di luar jam breakdown: tetap breakdown "AC HOT"; operator hanya ada di shift 2 tanggal 22
	op, ok := c.OperationalAt(&projCHR, &eqDT, at(24, 10, 0))
	if !ok || !op.BreakdownKnown || !op.IsBreakdown || op.BreakdownDescription == nil || *op.BreakdownDescription != "AC HOT" {
		t.Fatalf("breakdown menempel: %+v ok=%v", op, ok)
	}
	if !op.OperatorKnown || op.Operator != nil {
		t.Errorf("shift 1 tanggal 24 tidak punya operator: %+v", op)
	}
	if op, _ := c.OperationalAt(&projCHR, &eqDT, at(22, 23, 30)); op.Operator == nil || *op.Operator != "Dede" {
		t.Errorf("operator shift 2: %+v", op)
	}

	// status ready diinput → normal, deskripsi hilang
	st.mu.Lock()
	st.breakdown = append(st.breakdown, BreakdownRow{ProjectID: projCHR, OperationalDate: "2026-10-23", ShiftCode: "SHIFT_1", EquipmentCode: "DT10300", Status: "ready", UpdatedAt: at(23, 9, 0)})
	st.mu.Unlock()
	_ = c.Reload(context.Background(), PartBreakdowns)
	if op, _ := c.OperationalAt(&projCHR, &eqDT, at(24, 10, 0)); !op.BreakdownKnown || op.IsBreakdown || op.BreakdownDescription != nil {
		t.Errorf("setelah ready: %+v", op)
	}

	// unit tanpa catatan: breakdown "tidak diketahui" (nilai yang ada dibiarkan); unit/project tak dikenal: ok=false
	st.mu.Lock()
	st.breakdown = nil
	st.mu.Unlock()
	_ = c.Reload(context.Background(), PartBreakdowns)
	if op, ok := c.OperationalAt(&projCHR, &eqDT, at(24, 10, 0)); !ok || op.BreakdownKnown {
		t.Errorf("tanpa catatan: %+v ok=%v", op, ok)
	}
	other := uuid.New()
	if _, ok := c.OperationalAt(&projCHR, &other, at(24, 10, 0)); ok {
		t.Error("unit tak dikenal: ok seharusnya false")
	}
	if _, ok := c.OperationalAt(nil, &eqDT, at(24, 10, 0)); ok {
		t.Error("tanpa project: ok seharusnya false")
	}
}

// Muat ulang data induk (NOTIFY dari core) memanggil fungsi yang didaftarkan.
func TestOnReload(t *testing.T) {
	c := New(&fakeStore{})
	var got []string
	c.OnReload(func(parts []string) { got = append(got, parts...) })
	c.reloadLogged(context.Background(), "uji", PartBreakdowns, PartOperators)
	if len(got) != 2 || got[0] != PartBreakdowns || got[1] != PartOperators {
		t.Fatalf("hook menerima %v", got)
	}
}
