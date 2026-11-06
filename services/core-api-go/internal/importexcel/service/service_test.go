package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	activationdomain "simha/core-api/internal/activation/domain"
	activationsvc "simha/core-api/internal/activation/service"
	breakdowndomain "simha/core-api/internal/breakdown/domain"
	breakdownsvc "simha/core-api/internal/breakdown/service"
	operatordomain "simha/core-api/internal/operator/domain"
	operatorsvc "simha/core-api/internal/operator/service"
	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	shiftdomain "simha/core-api/internal/shift/domain"
	weighbridgedomain "simha/core-api/internal/weighbridge/domain"
	weighbridgesvc "simha/core-api/internal/weighbridge/service"
)

// Test ini memakai service ASLI fitur operator, activation, breakdown & weighbridge (pengurai baris
// tiap jenis), dengan database & sumber data project palsu di memori.

var projCHR = uuid.New()

// dailyMem = data palsu: sumber data project (project, shift, equipment, pilihan) + baris yang disimpan.
type dailyMem struct {
	equipment  map[string]uuid.UUID
	savedOps   []operatordomain.OperatorAssignment
	savedActs  []activationdomain.ActivationRow
	savedBds   []breakdowndomain.EquipmentBreakdown
	savedWbs   []weighbridgedomain.WeighbridgeTicket
	saveCalled int
}

var (
	shift1 = shiftdomain.Shift{ID: uuid.New(), Code: "SHIFT_1", Name: "Shift 1", Start: "07:00", End: "19:00", Sequence: 1}
	shift2 = shiftdomain.Shift{ID: uuid.New(), Code: "SHIFT_2", Name: "Shift 2", Start: "19:00", End: "07:00", Sequence: 2}
)

func (m *dailyMem) FindActive(_ context.Context, id uuid.UUID) (*projectdomain.Project, error) {
	return &projectdomain.Project{ID: id, Code: "CHR", Timezone: "Asia/Jakarta"}, nil
}
func (m *dailyMem) ListActive(context.Context, uuid.UUID) ([]shiftdomain.Shift, error) {
	return []shiftdomain.Shift{shift1, shift2}, nil
}
func (m *dailyMem) CodeMap(context.Context, uuid.UUID) (map[string]uuid.UUID, error) {
	return m.equipment, nil
}
func (m *dailyMem) List(context.Context, string) ([]referencedomain.ReferenceValue, error) {
	return []referencedomain.ReferenceValue{
		{GroupCode: referencedomain.RefBreakdownCategory, Code: "unscheduled_maintenance", Label: "Unsch Maintenance"},
		{GroupCode: referencedomain.RefBreakdownCategory, Code: "tyre_maintenance", Label: "Tyre Maintenance"},
		{GroupCode: referencedomain.RefRepairStatus, Code: "on_progress", Label: "On Progress"},
		{GroupCode: referencedomain.RefRepairStatus, Code: "done", Label: "Done"},
	}, nil
}

// Database palsu per fitur (hanya BulkUpsert yang dipakai impor; method lain tidak dipanggil).
type opStore struct {
	operatorsvc.Repository
	m *dailyMem
}

func (s opStore) BulkUpsert(_ context.Context, rows []operatordomain.OperatorAssignment, _ uuid.UUID, dry bool) (int, int, error) {
	s.m.saveCalled++
	if !dry {
		s.m.savedOps = rows
	}
	return len(rows), 0, nil
}

type actStore struct {
	activationsvc.Repository
	m *dailyMem
}

func (s actStore) BulkUpsert(_ context.Context, rows []activationdomain.ActivationRow, _ uuid.UUID, dry bool) (int, int, error) {
	s.m.saveCalled++
	if !dry {
		s.m.savedActs = rows
	}
	return len(rows), 0, nil
}

type bdStore struct {
	breakdownsvc.Repository
	m *dailyMem
}

func (s bdStore) BulkUpsert(_ context.Context, rows []breakdowndomain.EquipmentBreakdown, _ uuid.UUID, dry bool) (int, int, error) {
	s.m.saveCalled++
	if !dry {
		s.m.savedBds = rows
	}
	return len(rows), 0, nil
}

type wbStore struct {
	weighbridgesvc.Repository
	m *dailyMem
}

func (s wbStore) BulkUpsert(_ context.Context, rows []weighbridgedomain.WeighbridgeTicket, _ uuid.UUID, dry bool) (int, int, error) {
	s.m.saveCalled++
	if !dry {
		s.m.savedWbs = rows
	}
	return len(rows), 0, nil
}

// newImport merakit service impor seperti di cmd/api/main.go, tetapi dengan data palsu.
func newImport(st *dailyMem) *Service {
	daily := dailyinput.Loader{Projects: st, Shifts: st, Equipment: st, References: st}
	return New(daily, map[string]RowImporter{
		"operator-assignments":  operatorsvc.New(opStore{m: st}, daily),
		"equipment-activations": activationsvc.New(actStore{m: st}, nil),
		"equipment-breakdowns":  breakdownsvc.New(bdStore{m: st}, daily),
		"weighbridge-tickets":   weighbridgesvc.New(wbStore{m: st}, daily),
	})
}

// xlsx membuat file Excel: baris 1 judul, lalu baris data.
func xlsx(t *testing.T, rows ...[]any) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			f.SetCellValue("Sheet1", cell, v)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestImportOperator(t *testing.T) {
	ctx := context.Background()
	dt1 := uuid.New()
	st := &dailyMem{equipment: map[string]uuid.UUID{"DT10300": dt1}}
	s := newImport(st)
	disp := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleDispatcher}
	header := []any{"Tanggal", "Shift", "Asset ID", "Operator Name", "Description"}

	t.Run("file benar: shift berbagai bentuk, unit ditautkan bila ada di master, kunci sama → baris terakhir menang", func(t *testing.T) {
		file := xlsx(t, header,
			[]any{"22-10-2026", "Shift 1", "dt10300", "Budi", ""},
			[]any{"2026-10-22", "SHIFT_2", "DT99999", "Andi", "unit transporter"},
			[]any{"22/10/2026", "1", "DT10300", "Budi Santoso", ""}, // kunci sama dengan baris 2
		)
		res, err := s.Import(ctx, disp, "operator-assignments", nil, "op.xlsx", file, false)
		if err != nil {
			t.Fatalf("err=%v res=%+v", err, res)
		}
		if !res.Saved || res.TotalRows != 3 || res.DuplicateRows != 1 || res.ToCreate != 2 || len(res.Warnings) != 1 {
			t.Fatalf("hasil: %+v", res)
		}
		var budi, andi operatordomain.OperatorAssignment
		for _, o := range st.savedOps {
			if o.EquipmentCode == "DT10300" {
				budi = o
			} else {
				andi = o
			}
		}
		if budi.OperatorName != "Budi Santoso" || budi.EquipmentID == nil || *budi.EquipmentID != dt1 || budi.ShiftID != shift1.ID {
			t.Fatalf("baris DT10300: %+v", budi)
		}
		if andi.EquipmentID != nil || andi.ShiftID != shift2.ID {
			t.Fatalf("unit di luar master tetap disimpan tanpa tautan: %+v", andi)
		}
	})

	t.Run("ada baris salah: pratinjau menunjukkan nomor baris & kolom; simpan ditolak seluruhnya", func(t *testing.T) {
		rows := [][]any{header,
			{"32-10-2026", "Shift 1", "DT10300", "Budi", ""},  // baris 2: tanggal salah
			{"22-10-2026", "Shift 9", "DT10300", "Budi", ""},  // baris 3: shift tidak dikenal
			{"22-10-2026", "Shift 1", "DT10301", "", ""},      // baris 4: operator kosong
			{"22-10-2026", "Shift 1", "DT10302", "Citra", ""}, // baris 5: benar
		}
		before := st.saveCalled
		res, err := s.Import(ctx, disp, "operator-assignments", nil, "op.xlsx", xlsx(t, rows...), true)
		if err != nil || res.InvalidRows != 3 || res.ValidRows != 1 || res.Saved {
			t.Fatalf("pratinjau: %+v err=%v", res, err)
		}
		want := map[int]string{2: "Tanggal", 3: "Shift", 4: "Operator Name"}
		for _, e := range res.Errors {
			if want[e.Row] != e.Column {
				t.Errorf("kesalahan tak terduga: %+v", e)
			}
		}
		if _, err := s.Import(ctx, disp, "operator-assignments", nil, "op.xlsx", xlsx(t, rows...), false); !errors.Is(err, apperror.ErrImportHasErrors) {
			t.Fatalf("simpan seharusnya ditolak, err = %v", err)
		}
		if st.saveCalled != before {
			t.Fatal("tidak boleh ada yang disimpan bila ada baris salah")
		}
	})

	t.Run("viewer tidak boleh mengimpor", func(t *testing.T) {
		viewer := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleViewer}
		if _, err := s.Import(ctx, viewer, "operator-assignments", nil, "op.xlsx", xlsx(t, header, []any{"x"}), true); err != apperror.ErrForbidden {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestImportAktivasi(t *testing.T) {
	ctx := context.Background()
	dt1 := uuid.New()
	st := &dailyMem{equipment: map[string]uuid.UUID{"DT10300": dt1}}
	s := newImport(st)
	admin := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleAdmin}
	header := []any{"Date", "Equipment Code", "Status", "Description"}

	res, err := s.Import(ctx, admin, "equipment-activations", nil, "a.xlsx", xlsx(t, header,
		[]any{"22-10-2026", "DT10300", "Non Active", "servis"},
		[]any{"22-10-2026", "DT77777", "Active", ""}, // tidak ada di master
		[]any{"23-10-2026", "DT10300", "Aktif", ""},  // status tidak dikenal
	), true)
	if err != nil || res.InvalidRows != 2 || len(res.Errors) != 2 {
		t.Fatalf("hasil: %+v err=%v", res, err)
	}

	res, err = s.Import(ctx, admin, "equipment-activations", nil, "a.xlsx", xlsx(t, header,
		[]any{"22-10-2026", "dt10300", "non active", "servis"},
	), false)
	if err != nil || !res.Saved || st.savedActs[0].Status != activationdomain.ActivationInactive || st.savedActs[0].EquipmentID != dt1 {
		t.Fatalf("hasil: %+v err=%v saved=%+v", res, err, st.savedActs)
	}
}

func TestImportBreakdown(t *testing.T) {
	ctx := context.Background()
	st := &dailyMem{equipment: map[string]uuid.UUID{}}
	s := newImport(st)
	disp := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleDispatcher}
	header := []any{"Date", "Shift", "Asset ID", "Class", "Status", "Category", "Time Start", "Time End", "Duration",
		"Repair Status", "Description", "Location"}
	wib := time.FixedZone("WIB", 7*3600)

	res, err := s.Import(ctx, disp, "equipment-breakdowns", nil, "b.xlsx", xlsx(t, header,
		// baris 2: Shift 2 lewat tengah malam: 23:00 → 01:30 hari berikutnya, durasi dihitung 2.5 jam
		[]any{"22-10-2026", "Shift 2", "DT10300", "DUMP TRUCK", "Breakdown", "unsch maintenance", "23:00", "01:30", "", "On Progress", "", ""},
		// baris 3: status Continue tanpa jam selesai → selesai = akhir shift 1 (19:00); durasi Excel 0.375 hari = 9 jam cocok
		[]any{"22-10-2026", "Shift 1", "DT10301", "", "continue", "TYRE_MAINTENANCE", "10:00", "", 0.375, "done", "ban", "STA 24"},
		// baris 4: "Down" = sebutan lama untuk breakdown, tanpa jam
		[]any{"22-10-2026", "1", "DT10302", "", "Down", "Unsch Maintenance", "", "", "01:00", "Done", "", ""},
	), false)
	if err != nil || !res.Saved || len(st.savedBds) != 3 {
		t.Fatalf("hasil: %+v err=%v", res, err)
	}
	b := st.savedBds[0]
	if b.CategoryCode != "unscheduled_maintenance" || b.RepairStatusCode != "on_progress" ||
		!b.StartedAt.Equal(time.Date(2026, 10, 22, 23, 0, 0, 0, wib)) || !b.EndedAt.Equal(time.Date(2026, 10, 23, 1, 30, 0, 0, wib)) ||
		*b.DurationHours != 2.5 {
		t.Fatalf("baris 2: %+v", b)
	}
	b = st.savedBds[1]
	if b.Status != breakdowndomain.BreakdownStatusContinue || !b.EndedAt.Equal(time.Date(2026, 10, 22, 19, 0, 0, 0, wib)) || *b.DurationHours != 9 {
		t.Fatalf("baris 3: %+v", b)
	}
	b = st.savedBds[2]
	if b.Status != breakdowndomain.BreakdownStatusBreakdown || b.StartedAt != nil || *b.DurationHours != 1 {
		t.Fatalf("baris 4: %+v", b)
	}

	res, _ = s.Import(ctx, disp, "equipment-breakdowns", nil, "b.xlsx", xlsx(t, header,
		[]any{"22-10-2026", "Shift 1", "DT1", "", "Rusak", "Unsch Maintenance", "", "", "", "Done", "", ""},      // status salah
		[]any{"22-10-2026", "Shift 1", "DT1", "", "Ready", "Kategori X", "", "", "", "Done", "", ""},             // kategori salah
		[]any{"22-10-2026", "Shift 1", "DT1", "", "Ready", "Unsch Maintenance", "06:00", "", "", "Done", "", ""}, // di luar shift
		[]any{"22-10-2026", "Shift 1", "DT1", "", "Ready", "Unsch Maintenance", "08:00", "09:00", "02:00", "Done", "", ""},
		[]any{"22-10-2026", "Shift 1", "DT1", "", "Ready", "Unsch Maintenance", "", "", "", "", "", ""}, // repair status kosong
	), true)
	want := map[int]string{2: "Status", 3: "Category", 4: "Time Start", 5: "Duration", 6: "Repair Status"}
	if res.InvalidRows != 5 || len(res.Errors) != 5 {
		t.Fatalf("hasil: %+v", res)
	}
	for _, e := range res.Errors {
		if want[e.Row] != e.Column {
			t.Errorf("kesalahan tak terduga: %+v", e)
		}
	}
}

func TestImportWeighbridge(t *testing.T) {
	ctx := context.Background()
	st := &dailyMem{equipment: map[string]uuid.UUID{}}
	s := newImport(st)
	disp := access.Actor{UserID: uuid.New(), ProjectID: &projCHR, Role: access.RoleDispatcher}
	header := []any{"Date", "Shift", "Ticket No", "Asset ID", "Product", "Gross", "Tare", "Net", "Recipient", "Customer",
		"Transporter", "Gross Time", "Tare Time", "Gross Operator", "Tare Operator", "Description", "Location"}
	wib := time.FixedZone("WIB", 7*3600)
	row := func(ticket string, gross, tare, net any, tareTime string) []any {
		return []any{"22-10-2026", "Shift 2", ticket, "10419", "BATUBARA", gross, tare, net, "PT. RMKO", "PT. DBU", "PT. RMKO",
			"23:50", tareTime, "Yulius", "Tegar", "", ""}
	}

	res, err := s.Import(ctx, disp, "weighbridge-tickets", nil, "w.xlsx", xlsx(t, header,
		row("T1", 46200, 14680, "", "00:10"),          // net kosong → dihitung; tare lewat tengah malam
		row("T2", "46,200", "14,680", 31520, "00:20"), // ribuan dengan koma (seperti lama)
	), false)
	if err != nil || !res.Saved {
		t.Fatalf("hasil: %+v err=%v", res, err)
	}
	w := st.savedWbs[0]
	if w.NetKg != 31520 || !w.TareAt.Equal(time.Date(2026, 10, 23, 0, 10, 0, 0, wib)) || !w.GrossAt.Equal(time.Date(2026, 10, 22, 23, 50, 0, 0, wib)) {
		t.Fatalf("T1: %+v", w)
	}
	if st.savedWbs[1].GrossKg != 46200 || st.savedWbs[1].NetKg != 31520 {
		t.Fatalf("T2: %+v", st.savedWbs[1])
	}

	res, _ = s.Import(ctx, disp, "weighbridge-tickets", nil, "w.xlsx", xlsx(t, header,
		row("T3", 46200, 14680, 30000, ""), // net beda
		row("T4", 10000, 14680, "", ""),    // tare > gross
		row("T5", "abc", 14680, "", ""),    // gross bukan angka
		row("", 46200, 14680, "", ""),      // nomor tiket kosong
	), true)
	want := map[int]string{2: "Net", 3: "Tare", 4: "Gross", 5: "Ticket No"}
	if res.InvalidRows != 4 || len(res.Errors) != 4 {
		t.Fatalf("hasil: %+v", res)
	}
	for _, e := range res.Errors {
		if want[e.Row] != e.Column {
			t.Errorf("kesalahan tak terduga: %+v", e)
		}
	}
}
