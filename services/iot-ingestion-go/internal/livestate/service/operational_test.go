package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// fakeResolver = data induk palsu: jawaban bisa diganti di tengah tes (mirip breakdown diinput / diubah ke ready).
type fakeResolver struct {
	op    Operational
	known bool
}

func (f *fakeResolver) OperationalAt(_, _ *uuid.UUID, _ time.Time) (Operational, bool) {
	return f.op, f.known
}

func sp(s string) *string { return &s }

// Breakdown diinput / diubah ke ready → kondisi terakhir unit (termasuk yang offline) berubah seketika, dan HANYA karena itu.
func TestRefreshOperational(t *testing.T) {
	ctx := context.Background()
	project, eq := uuid.New(), uuid.New()
	t0 := time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)
	offline := domain.StatusOffline
	no := false
	db := &fakeDB{rows: map[uuid.UUID]domain.LiveState{
		eq: {EquipmentID: eq, ProjectID: &project, EventTime: t0, EquipmentStatus: &offline, IsBreakdown: &no},
	}}
	c := New(db)
	if _, err := c.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := &fakeResolver{known: true, op: Operational{BreakdownKnown: true, OperatorKnown: true}}
	now := t0.Add(time.Hour)

	// tidak ada perubahan: tidak menulis ke database
	writes := db.writes
	if ch, err := c.RefreshOperational(ctx, r, now); err != nil || len(ch) != 0 || db.writes != writes {
		t.Fatalf("tanpa perubahan: ch=%v err=%v tulis=%d", ch, err, db.writes-writes)
	}

	// breakdown diinput: is_breakdown true + deskripsi "AC HOT", operator ikut
	r.op = Operational{BreakdownKnown: true, IsBreakdown: true, BreakdownDescription: sp("AC HOT"), OperatorKnown: true, Operator: sp("Dede Jumari")}
	ch, err := c.RefreshOperational(ctx, r, now)
	if err != nil || len(ch) != 1 {
		t.Fatalf("breakdown diinput: ch=%v err=%v", ch, err)
	}
	s, _ := c.Get(eq)
	if s.IsBreakdown == nil || !*s.IsBreakdown || s.BreakdownDescription == nil || *s.BreakdownDescription != "AC HOT" || *s.OperatorName != "Dede Jumari" {
		t.Fatalf("memori: %+v", s)
	}
	if *db.rows[eq].BreakdownDescription != "AC HOT" {
		t.Fatal("database harus ikut berubah")
	}
	if *s.EquipmentStatus != domain.StatusOffline {
		t.Error("status unit tidak boleh berubah oleh pembaruan ini")
	}

	// menempel: berjam-jam kemudian (jawaban data induk sama) tidak ada perubahan, tidak menulis ulang
	writes = db.writes
	if ch, _ := c.RefreshOperational(ctx, r, now.Add(48*time.Hour)); len(ch) != 0 || db.writes != writes {
		t.Error("status breakdown tidak boleh berubah karena waktu")
	}

	// unit belum punya catatan breakdown: nilai yang ada dibiarkan, operator tetap mengikuti shift
	r.op = Operational{BreakdownKnown: false, OperatorKnown: true, Operator: sp("Dede Jumari")}
	if ch, _ := c.RefreshOperational(ctx, r, now); len(ch) != 0 {
		t.Error("tanpa catatan breakdown: nilai yang ada dibiarkan")
	}
	if s, _ = c.Get(eq); !*s.IsBreakdown {
		t.Error("breakdown tidak boleh hilang tanpa status ready")
	}

	// status ready diinput: normal, deskripsi hilang
	r.op = Operational{BreakdownKnown: true, IsBreakdown: false, OperatorKnown: true, Operator: sp("Dede Jumari")}
	if ch, _ := c.RefreshOperational(ctx, r, now); len(ch) != 1 {
		t.Fatalf("ready: ch=%v", ch)
	}
	s, _ = c.Get(eq)
	if *s.IsBreakdown || s.BreakdownDescription != nil {
		t.Fatalf("setelah ready: %v %v", *s.IsBreakdown, s.BreakdownDescription)
	}

	// project/unit belum diketahui: semuanya dibiarkan
	r.known, r.op = false, Operational{BreakdownKnown: true, IsBreakdown: true}
	if ch, _ := c.RefreshOperational(ctx, r, now); len(ch) != 0 {
		t.Error("data induk belum ada: tidak boleh mengubah")
	}

	// database gagal: memori tetap sama dengan database
	r.known = true
	db.failing = true
	if _, err := c.RefreshOperational(ctx, r, now); err == nil {
		t.Fatal("galat database harus dilaporkan")
	}
	if s, _ = c.Get(eq); *s.IsBreakdown {
		t.Error("database gagal: memori tidak boleh berubah")
	}
}
