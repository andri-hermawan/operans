package main

import (
	"encoding/json"
	"math"
	"testing"

	"simha/iot-ingestion/internal/ingest/domain"
)

// Seluruh skenario dijalankan lewat aturan fuel server (unit PERCENTAGE, tangki 210 L): setiap data harus punya
// raw setara, liter, persen, dan selisih; dua pengisian BBM (data ke-10 dan ke-11) terdeteksi sebagai kenaikan.
func TestSkenarioLewatAturanFuel(t *testing.T) {
	var sk []langkah
	_ = json.Unmarshal(skenarioJSON, &sk)
	cal := &domain.Calibration{Type: domain.CalibrationPercentage, CapacityLiters: 210, Points: []domain.CalPoint{{Sensor: 0, Liters: 0}, {Sensor: 100, Liters: 210}}}

	var prev *domain.LiveState
	naik, turunBesar := 0, 0
	for i, s := range sk {
		p := s.FuelPercent
		e := &domain.Event{FuelSensorPercent: &p}
		if !domain.ApplyFuel(prev, cal, e) {
			t.Fatalf("data #%d: bacaan sah seharusnya diterima", i+1)
		}
		if e.FuelLevelRaw == nil || e.FuelVolumeLiter == nil || e.FuelPercent == nil || e.FuelDeltaLiter == nil {
			t.Fatalf("data #%d: kolom fuel wajib belum terisi", i+1)
		}
		if *e.FuelDeltaLiter > 20 {
			naik++
		}
		if *e.FuelDeltaLiter < -2 {
			turunBesar++
		}
		prev = &domain.LiveState{}
		prev.FuelLevelRaw, prev.FuelVolumeLiter, prev.FuelPercent = e.FuelLevelRaw, e.FuelVolumeLiter, e.FuelPercent
	}
	if naik != 2 {
		t.Errorf("pengisian BBM terdeteksi %d kali, seharusnya 2", naik)
	}
	if turunBesar != 0 {
		t.Errorf("ada %d penurunan > 2 L antar data: skenario normal hanya turun pelan", turunBesar)
	}
	if got := *prev.FuelVolumeLiter; math.Abs(got-167.37) > 0.5 { // 79,7% × 210 L
		t.Errorf("liter akhir = %v", got)
	}
}
