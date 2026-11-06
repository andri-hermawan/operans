package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
	livestatesvc "simha/iot-ingestion/internal/livestate/service"
)

// memPersister = penyimpan live state palsu untuk service.Cache (fitur livestate) yang asli.
type memPersister struct {
	mu   sync.Mutex
	rows map[uuid.UUID]domain.LiveState
}

func (m *memPersister) LoadLiveStates(context.Context) ([]domain.LiveState, error) { return nil, nil }
func (m *memPersister) UpsertLiveState(_ context.Context, s domain.LiveState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[s.EquipmentID] = s
	return nil
}

// zoneEnricher = data induk palsu: zona ditentukan dari "lintang" event (lat = fid zona, 0 = di luar).
type zoneEnricher struct{}

func (zoneEnricher) Enrich(_, _ *uuid.UUID, e *domain.Event) {
	if e.Latitude == nil {
		return
	}
	fid := int32(*e.Latitude)
	inside := fid != 0
	e.GeofenceFID, e.IsInsideGeofence = &fid, &inside
	if inside {
		id := uuid.NewSHA1(uuid.Nil, []byte{byte(fid)})
		name, cat := fmt.Sprintf("Km %d", fid), "Trase Utama"
		e.ZoneID, e.SegmentName, e.LocationCategory = &id, &name, &cat
	}
}

func TestHandleStatusMuatanDanZona(t *testing.T) {
	store := &fakeStore{saved: map[string]bool{}}
	live := livestatesvc.New(&memPersister{rows: map[uuid.UUID]domain.LiveState{}})
	bus := &fakeBus{}
	s := NewIngest(store, live, bus).WithEnricher(zoneEnricher{})
	s.now = func() time.Time { return now }
	project := uuid.New()
	d := *dev
	d.ProjectID = &project

	kirim := func(id string, menit int, fid float64, speed float64, engine bool) Result {
		t.Helper()
		lon, gsm, op := 103.8, int16(4), int32(51010)
		e := &domain.Event{EventID: id, EventTime: now.Add(time.Duration(menit-60) * time.Minute)}
		e.Latitude, e.Longitude, e.SpeedKph, e.IsEngineOn = &fid, &lon, &speed, &engine
		e.GSMSignalLevel, e.GSMOperatorCode = &gsm, &op
		res, err := s.Handle(context.Background(), &d, e)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	vessel := func() string {
		st, _ := live.Get(equipmentID)
		if st.VesselStatus == nil {
			return "<nil>"
		}
		return *st.VesselStatus
	}

	// rute: zona 5 → 4 → 3 (fid turun = bermuatan), keluar zona, berhenti di luar, kembali ke 4 (kosong)
	kirim("1", 0, 5, 30, true)
	if v := vessel(); v != domain.VesselEmpty {
		t.Fatalf("data pertama: %s", v)
	}
	kirim("2", 1, 4, 30, true)
	if v := vessel(); v != domain.VesselLoaded {
		t.Fatalf("5→4: vessel=%s", v)
	}
	kirim("3", 2, 3, 30, true)
	kirim("4", 3, 0, 30, true) // keluar jalur sambil jalan → status dipertahankan
	if v := vessel(); v != domain.VesselLoaded {
		t.Fatalf("keluar zona: vessel=%s", v)
	}
	kirim("5", 4, 0, 0, false) // mesin mati di luar zona → unknown
	if v := vessel(); v != domain.VesselUnknown {
		t.Fatalf("stop di luar zona: %s", v)
	}
	kirim("6", 5, 0, 10, true) // jalan lagi di luar zona → kembali ke status zona terakhir (loaded)
	if v := vessel(); v != domain.VesselLoaded {
		t.Fatalf("jalan lagi di luar: %s", v)
	}
	kirim("7", 6, 4, 30, true) // masuk zona 4 (fid naik dari 3) → empty
	if v := vessel(); v != domain.VesselEmpty {
		t.Fatalf("masuk 4: vessel=%s", v)
	}

	// data terlambat: tersimpan dengan zona, tanpa status muatan, tanpa kejadian, kondisi terakhir tetap
	r := kirim("telat", -30, 9, 30, true)
	if len(r.Geofence) != 0 || r.LiveState != nil || vessel() != domain.VesselEmpty {
		t.Fatalf("data terlambat: %+v vessel=%s", r, vessel())
	}
	// rute hanya segmen jalan (ORIG_FID ≠ 0) → tidak ada kejadian area (aturan geofence lama)
	if len(store.geofence) != 0 {
		t.Fatalf("kejadian area %d, mau 0: %+v", len(store.geofence), store.geofence)
	}
	_ = bus
}

func (zoneEnricher) Calibration(*uuid.UUID) *domain.Calibration  { return nil }
func (zoneEnricher) Rules(*uuid.UUID, *string) domain.AlertRules { return domain.DefaultAlertRules }
