package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/repository"
)

// percentStore mencatat event yang BENAR-BENAR disimpan (setelah diperkaya & dihitung).
type percentStore struct {
	*fakeStore
	last domain.Event
}

func (p *percentStore) InsertEvent(ctx context.Context, d *devicedomain.Device, e *domain.Event) (repository.Saved, bool, error) {
	p.last = *e
	return p.fakeStore.InsertEvent(ctx, d, e)
}

// percentEnricher = data induk palsu: unit bertipe percentage, tangki 210 L, titik lurus 0..100 % → 0..210 L.
type percentEnricher struct{ fakeEnricher }

func (*percentEnricher) Calibration(*uuid.UUID) *domain.Calibration {
	return &domain.Calibration{Type: domain.CalibrationPercentage, CapacityLiters: 210, Points: []domain.CalPoint{{Sensor: 0, Liters: 0}, {Sensor: 100, Liters: 210}}}
}

// Unit PERCENTAGE: dari persen ECU, kolom fuel yang wajib (raw, liter, persen, selisih) terisi saat disimpan.
func TestHandleFuelPercentage(t *testing.T) {
	store := &percentStore{fakeStore: &fakeStore{saved: map[string]bool{}}}
	s := NewIngest(store, nil, nil)
	s.now = func() time.Time { return now }
	s.enrich = &percentEnricher{}

	pct := 50.0
	e := event("pct-1")
	e.FuelSensorPercent = &pct
	if _, err := s.Handle(context.Background(), dev, e); err != nil {
		t.Fatal(err)
	}
	got := store.last
	if got.FuelLevelRaw == nil || got.FuelVolumeLiter == nil || got.FuelPercent == nil || got.FuelDeltaLiter == nil {
		t.Fatalf("kolom fuel wajib terisi: raw=%v liter=%v persen=%v selisih=%v", got.FuelLevelRaw, got.FuelVolumeLiter, got.FuelPercent, got.FuelDeltaLiter)
	}
	if *got.FuelLevelRaw != 2047.5 || *got.FuelVolumeLiter != 105 || *got.FuelPercent != 50 || *got.FuelDeltaLiter != 0 {
		t.Errorf("raw=%v liter=%v persen=%v selisih=%v", *got.FuelLevelRaw, *got.FuelVolumeLiter, *got.FuelPercent, *got.FuelDeltaLiter)
	}
	if got.FuelTemperatureC != nil {
		t.Error("ECU tidak mengirim suhu: kolom suhu harus kosong")
	}
}
