package domain

import (
	"errors"
	"testing"
	"time"
)

// ptr membuat pointer dari sebuah nilai (Go tidak bisa menulis &5.0 langsung).
func ptr[T any](v T) *T { return &v }

func validEvent(now time.Time) Event {
	return Event{
		EventID:   "463719065679574-1792656000000",
		IMEI:      "463719065679574",
		EventTime: now.Add(-time.Minute),
		Position:  Position{Latitude: ptr(-1.2654), Longitude: ptr(116.8312), SpeedKph: ptr(32.5)},
	}
}

func TestValidate(t *testing.T) {
	now := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		ubah    func(e *Event)
		wantErr bool
	}{
		{"event valid", func(e *Event) {}, false},
		{"event_id kosong", func(e *Event) { e.EventID = "" }, true},
		{"imei 30 digit", func(e *Event) { e.IMEI = "463719065679574463719065679574" }, true},
		{"jam alat 1970", func(e *Event) { e.EventTime = time.Unix(0, 0) }, true},
		{"waktu masa depan", func(e *Event) { e.EventTime = now.Add(time.Hour) }, true},
		{"latitude tanpa longitude", func(e *Event) { e.Longitude = nil }, true},
		{"speed negatif", func(e *Event) { e.SpeedKph = ptr(-5.0) }, true},
		{"fuel_percent 150", func(e *Event) { e.FuelPercent = ptr(150.0) }, true},
		{"data terlambat 3 hari tetap diterima", func(e *Event) { e.EventTime = now.Add(-72 * time.Hour) }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := validEvent(now)
			tt.ubah(&e)
			err := e.Validate(now)
			if tt.wantErr && !errors.Is(err, ErrInvalid) {
				t.Fatalf("seharusnya ErrInvalid, didapat: %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("seharusnya valid, didapat: %v", err)
			}
		})
	}
}

func TestNormalizePosisiNol(t *testing.T) {
	e := Event{Position: Position{Latitude: ptr(0.0), Longitude: ptr(0.0)}}
	e.Normalize()
	if e.Latitude != nil || e.Longitude != nil {
		t.Fatal("posisi 0,0 seharusnya dianggap tidak ada (nil)")
	}
	if e.Metadata == nil {
		t.Fatal("metadata seharusnya diisi map kosong")
	}
}
