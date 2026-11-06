package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// fakeDB = database palsu: menghitung berapa kali ditulis dan dibaca.
type fakeDB struct {
	mu      sync.Mutex
	rows    map[uuid.UUID]domain.LiveState
	reads   int
	writes  int
	failing bool
}

func (f *fakeDB) LoadLiveStates(context.Context) ([]domain.LiveState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	out := []domain.LiveState{}
	for _, s := range f.rows {
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeDB) UpsertLiveState(_ context.Context, s domain.LiveState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing {
		return errors.New("database mati")
	}
	f.writes++
	f.rows[s.EquipmentID] = s
	return nil
}

func TestCache(t *testing.T) {
	ctx := context.Background()
	projectA, projectB := uuid.New(), uuid.New()
	eqA, eqB := uuid.New(), uuid.New()
	t0 := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC)

	// Database sudah berisi kondisi terakhir equipment A (mis. sebelum server di-restart)
	db := &fakeDB{rows: map[uuid.UUID]domain.LiveState{
		eqA: {EquipmentID: eqA, ProjectID: &projectA, TelemetryEventID: 1, EventTime: t0},
	}}
	c := New(db)
	if n, err := c.Load(ctx); err != nil || n != 1 {
		t.Fatalf("load: n=%d err=%v", n, err)
	}

	// 100 data baru untuk A → database DITULIS 100 kali, tapi TIDAK PERNAH dibaca lagi
	for i := 1; i <= 100; i++ {
		e := domain.Event{EventTime: t0.Add(time.Duration(i) * time.Second)}
		if _, changed, err := c.Apply(ctx, domain.Owner{EquipmentID: eqA, ProjectID: &projectA}, &e, int64(i+1), t0); err != nil || !changed {
			t.Fatalf("apply %d: changed=%v err=%v", i, changed, err)
		}
	}
	if db.reads != 1 || db.writes != 100 {
		t.Fatalf("database dibaca %d kali (harus 1), ditulis %d kali (harus 100)", db.reads, db.writes)
	}
	if s, _ := c.Get(eqA); s.TelemetryEventID != 101 {
		t.Fatalf("kondisi terakhir A seharusnya event 101, dapat %d", s.TelemetryEventID)
	}

	// Data terlambat → tidak ditulis
	late := domain.Event{EventTime: t0}
	if _, changed, _ := c.Apply(ctx, domain.Owner{EquipmentID: eqA, ProjectID: &projectA}, &late, 999, t0); changed || db.writes != 100 {
		t.Fatal("data terlambat tidak boleh menimpa kondisi terbaru")
	}

	// Equipment B di project lain
	e := domain.Event{EventTime: t0}
	c.Apply(ctx, domain.Owner{EquipmentID: eqB, ProjectID: &projectB}, &e, 500, t0)
	if len(c.List(&projectA)) != 1 || len(c.List(&projectB)) != 1 || len(c.List(nil)) != 2 {
		t.Fatal("List harus memisahkan per project (nil = semua)")
	}

	// Database gagal → memori TIDAK berubah (memori & database tetap sama)
	db.failing = true
	e2 := domain.Event{EventTime: t0.Add(time.Hour)}
	if _, _, err := c.Apply(ctx, domain.Owner{EquipmentID: eqB, ProjectID: &projectB}, &e2, 501, t0); err == nil {
		t.Fatal("seharusnya error")
	}
	if s, _ := c.Get(eqB); s.TelemetryEventID != 500 {
		t.Fatal("memori tidak boleh berubah bila database gagal")
	}
}

// Banyak data bersamaan untuk equipment yang sama (mis. 2 alat di 1 truk) tetap berurutan.
func TestCacheBersamaan(t *testing.T) {
	db := &fakeDB{rows: map[uuid.UUID]domain.LiveState{}}
	c := New(db)
	eq := uuid.New()
	t0 := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e := domain.Event{EventTime: t0.Add(time.Duration(i) * time.Second)}
			unlock := c.Lock(eq) // seperti service.Ingest: kunci per equipment selama memproses
			defer unlock()
			c.Apply(context.Background(), domain.Owner{EquipmentID: eq}, &e, int64(i), t0)
		}(i)
	}
	wg.Wait()
	if s, _ := c.Get(eq); s.TelemetryEventID != 50 {
		t.Fatalf("kondisi akhir harus event terbaru (50), dapat %d", s.TelemetryEventID)
	}
}

func TestMarkOffline(t *testing.T) {
	db := &fakeDB{rows: map[uuid.UUID]domain.LiveState{}}
	c := New(db)
	now := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
	running := domain.StatusRunning
	diam, aktif := uuid.New(), uuid.New()
	c.states[diam] = domain.LiveState{EquipmentID: diam, ReceivedAt: now.Add(-11 * time.Minute), EquipmentStatus: &running}
	c.states[aktif] = domain.LiveState{EquipmentID: aktif, ReceivedAt: now.Add(-2 * time.Minute), EquipmentStatus: &running}

	changed, err := c.MarkOffline(context.Background(), now, 10*time.Minute)
	if err != nil || len(changed) != 1 || changed[0].EquipmentID != diam {
		t.Fatalf("changed=%+v err=%v", changed, err)
	}
	if s, _ := c.Get(diam); *s.EquipmentStatus != domain.StatusOffline || db.rows[diam].EquipmentStatus == nil {
		t.Fatal("unit diam harus offline di memori & database")
	}
	if s, _ := c.Get(aktif); *s.EquipmentStatus != domain.StatusRunning {
		t.Fatal("unit aktif tidak boleh berubah")
	}
	// pemeriksaan berikutnya: yang sudah offline tidak disiarkan ulang
	if changed, _ := c.MarkOffline(context.Background(), now.Add(time.Minute), 10*time.Minute); len(changed) != 0 {
		t.Fatalf("sudah offline, tidak perlu disiarkan lagi: %+v", changed)
	}
}
