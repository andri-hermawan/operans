package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/repository"
)

// fakeStore = database palsu: mengingat event_id yang sudah disimpan, bisa disuruh "mati".
type fakeStore struct {
	saved    map[string]bool
	dbDown   bool
	geofence []domain.GeofenceEvent
	alerts   []domain.Alert
	fuel     []domain.FuelEvent
}

func (f *fakeStore) InsertGeofenceEvent(_ context.Context, g *domain.GeofenceEvent) error {
	f.geofence = append(f.geofence, *g)
	return nil
}

func (f *fakeStore) InsertEvent(_ context.Context, _ *devicedomain.Device, e *domain.Event) (repository.Saved, bool, error) {
	if f.dbDown {
		return repository.Saved{}, false, errors.New("koneksi database putus")
	}
	if f.saved[e.EventID] {
		return repository.Saved{}, false, nil // duplikat
	}
	f.saved[e.EventID] = true
	return repository.Saved{ID: int64(len(f.saved)), ReceivedAt: now}, true, nil
}

// fakeLive = penyimpan kondisi terakhir palsu: mencatat event apa saja yang diterapkan.
type fakeLive struct {
	applied []int64
	failing bool
}

func (f *fakeLive) Lock(uuid.UUID) func()                  { return func() {} }
func (f *fakeLive) Get(uuid.UUID) (domain.LiveState, bool) { return domain.LiveState{}, false }
func (f *fakeLive) Apply(_ context.Context, owner domain.Owner, e *domain.Event, id int64, _ time.Time) (domain.LiveState, bool, error) {
	if f.failing {
		return domain.LiveState{}, false, errors.New("gagal simpan kondisi terakhir")
	}
	f.applied = append(f.applied, id)
	return domain.LiveState{EquipmentID: owner.EquipmentID, TelemetryEventID: id, EquipmentStatus: e.EquipmentStatus}, true, nil
}

// fakeBus = penyiar palsu: mencatat jenis pesan yang disiarkan.
type fakeBus struct{ types []string }

func (f *fakeBus) Publish(_ *uuid.UUID, msgType string, _ any) { f.types = append(f.types, msgType) }

var (
	now         = time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	equipmentID = uuid.New()
	dev         = &devicedomain.Device{ID: uuid.New(), IMEI: "000000000000001", EquipmentID: &equipmentID}
)

func newTestIngest(store *fakeStore) *Ingest {
	s := NewIngest(store, nil, nil)
	s.now = func() time.Time { return now } // waktu "tetap" agar hasil test selalu sama
	return s
}

func event(id string) *domain.Event {
	return &domain.Event{EventID: id, EventTime: now.Add(-time.Minute)}
}

func TestHandle(t *testing.T) {
	t.Run("event baru tersimpan, IMEI diisi dari alat yang login", func(t *testing.T) {
		e := event("a")
		res, err := newTestIngest(&fakeStore{saved: map[string]bool{}}).Handle(context.Background(), dev, e)
		if err != nil || !res.Inserted || res.Duplicate {
			t.Fatalf("res=%+v err=%v", res, err)
		}
		if e.IMEI != dev.IMEI {
			t.Fatalf("IMEI seharusnya diisi %s, dapat %s", dev.IMEI, e.IMEI)
		}
	})

	t.Run("kirim ulang = duplikat, bukan error", func(t *testing.T) {
		s := newTestIngest(&fakeStore{saved: map[string]bool{}})
		s.Handle(context.Background(), dev, event("b"))
		res, err := s.Handle(context.Background(), dev, event("b"))
		if err != nil || !res.Duplicate {
			t.Fatalf("res=%+v err=%v", res, err)
		}
	})

	t.Run("IMEI berbeda dari alat yang login ditolak", func(t *testing.T) {
		e := event("c")
		e.IMEI = "463719065679574"
		_, err := newTestIngest(&fakeStore{saved: map[string]bool{}}).Handle(context.Background(), dev, e)
		if !IsInvalid(err) {
			t.Fatalf("seharusnya ErrInvalid, dapat %v", err)
		}
	})

	t.Run("data tidak valid tidak sampai ke database", func(t *testing.T) {
		store := &fakeStore{saved: map[string]bool{}}
		_, err := newTestIngest(store).Handle(context.Background(), dev, event(""))
		if !IsInvalid(err) || len(store.saved) != 0 {
			t.Fatalf("err=%v tersimpan=%d", err, len(store.saved))
		}
	})

	t.Run("database mati = error SERVER, bukan error data", func(t *testing.T) {
		_, err := newTestIngest(&fakeStore{dbDown: true}).Handle(context.Background(), dev, event("d"))
		if err == nil || IsInvalid(err) {
			t.Fatalf("seharusnya error server, dapat %v", err)
		}
	})
}

func TestHandleKondisiTerakhir(t *testing.T) {
	newWithLive := func(live *fakeLive) *Ingest {
		s := newTestIngest(&fakeStore{saved: map[string]bool{}})
		s.live = live
		return s
	}

	t.Run("event baru: status dihitung, kondisi terakhir diperbarui & disiarkan", func(t *testing.T) {
		live := &fakeLive{}
		bus := &fakeBus{}
		s := newWithLive(live)
		s.bus = bus
		res, err := s.Handle(context.Background(), dev, event("e"))
		if err != nil || res.LiveState == nil || len(live.applied) != 1 {
			t.Fatalf("res=%+v err=%v applied=%v", res, err, live.applied)
		}
		if len(bus.types) != 1 || bus.types[0] != "equipment_state" {
			t.Fatalf("siaran = %v", bus.types)
		}
		// Tanpa sinyal GSM → offline (aturan lama)
		if *res.LiveState.EquipmentStatus != domain.StatusOffline {
			t.Fatalf("status = %s", *res.LiveState.EquipmentStatus)
		}
	})

	t.Run("duplikat tidak mengubah kondisi terakhir", func(t *testing.T) {
		live := &fakeLive{}
		s := newWithLive(live)
		s.Handle(context.Background(), dev, event("f"))
		s.Handle(context.Background(), dev, event("f"))
		if len(live.applied) != 1 {
			t.Fatalf("diterapkan %d kali, seharusnya 1", len(live.applied))
		}
	})

	t.Run("gagal simpan kondisi terakhir tidak menggagalkan data utama", func(t *testing.T) {
		res, err := newWithLive(&fakeLive{failing: true}).Handle(context.Background(), dev, event("g"))
		if err != nil || !res.Inserted {
			t.Fatalf("data utama seharusnya tetap tersimpan: res=%+v err=%v", res, err)
		}
	})

	t.Run("alat tanpa equipment dilewati", func(t *testing.T) {
		live := &fakeLive{}
		tanpaEquipment := &devicedomain.Device{ID: uuid.New(), IMEI: dev.IMEI}
		newWithLive(live).Handle(context.Background(), tanpaEquipment, event("h"))
		if len(live.applied) != 0 {
			t.Fatal("alat tanpa equipment tidak punya kondisi terakhir")
		}
	})
}

// fakeEnricher = data induk palsu: mengisi shift & operator, mencatat project/equipment yang diminta.
type fakeEnricher struct{ gotProject, gotEquipment *uuid.UUID }

func (f *fakeEnricher) Enrich(projectID, equipmentID *uuid.UUID, e *domain.Event) {
	f.gotProject, f.gotEquipment = projectID, equipmentID
	date, shift, op := "2026-10-22", "SHIFT_1", "Budi"
	e.OperationalDate, e.ShiftCode, e.OperatorName = &date, &shift, &op
}

func TestHandlePengayaan(t *testing.T) {
	en := &fakeEnricher{}
	s := newTestIngest(&fakeStore{saved: map[string]bool{}}).WithEnricher(en)
	e := event("c1")
	if _, err := s.Handle(context.Background(), dev, e); err != nil {
		t.Fatal(err)
	}
	if en.gotEquipment != dev.EquipmentID || e.ShiftCode == nil || *e.OperatorName != "Budi" {
		t.Fatalf("pengayaan tidak dipanggil dengan benar: %+v", e.Derived)
	}
	// event tidak valid tidak diperkaya
	en.gotEquipment = nil
	bad := event("c2")
	bad.EventTime = now.Add(time.Hour)
	if _, err := s.Handle(context.Background(), dev, bad); err == nil || en.gotEquipment != nil {
		t.Fatal("event ditolak seharusnya tidak diperkaya")
	}
}

func (f *fakeStore) InsertAlert(_ context.Context, a *domain.Alert) error {
	f.alerts = append(f.alerts, *a)
	a.ID = int64(len(f.alerts))
	return nil
}

func (f *fakeStore) InsertFuelEvent(_ context.Context, fe *domain.FuelEvent) error {
	f.fuel = append(f.fuel, *fe)
	return nil
}

func (f *fakeEnricher) Calibration(*uuid.UUID) *domain.Calibration  { return nil }
func (f *fakeEnricher) Rules(*uuid.UUID, *string) domain.AlertRules { return domain.DefaultAlertRules }

func (f *fakeStore) CloseAlert(_ context.Context, eq uuid.UUID, typ string, end time.Time) (*domain.Alert, error) {
	for i := range f.alerts {
		if f.alerts[i].AlertType == typ && f.alerts[i].EndedAt == nil {
			f.alerts[i].EndedAt = &end
			return &f.alerts[i], nil
		}
	}
	return nil, nil
}

func (f *fakeStore) ExtendAlert(_ context.Context, _ uuid.UUID, typ string, start, end time.Time) (*domain.Alert, error) {
	for i := range f.alerts {
		if f.alerts[i].AlertType == typ && f.alerts[i].StartedAt.Equal(start) {
			f.alerts[i].EndedAt = &end
			return &f.alerts[i], nil
		}
	}
	return nil, nil
}
