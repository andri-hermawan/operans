package domain

import (
	"testing"
	"time"
)

// kalibrasi lurus seperti data lama: 0..4095 → 0..210 L
var cal210 = &Calibration{Type: "lls", CapacityLiters: 210, Points: []CalPoint{{0, 0}, {4095, 210}}}

func TestCalibrationLiters(t *testing.T) {
	l, pct, ok := cal210.Liters(2047.5)
	if !ok || l != 105 || pct != 50 {
		t.Fatalf("%v %v %v", l, pct, ok)
	}
	if _, _, ok := cal210.Liters(5000); ok {
		t.Fatal("di luar rentang seharusnya tidak dihitung")
	}
}

func TestApplyFuel(t *testing.T) {
	raw := func(v float64) *Event { return &Event{Fuel: Fuel{FuelLevelRaw: &v}} }
	prev := &LiveState{}
	pv, pl := 1950.0, 100.0
	prev.FuelLevelRaw, prev.FuelVolumeLiter = &pv, &pl

	e := raw(1755) // 90 L
	if !ApplyFuel(prev, cal210, e) || *e.FuelVolumeLiter != 90 || *e.FuelDeltaLiter != -10 {
		t.Fatalf("bacaan normal: %v %v", *e.FuelVolumeLiter, *e.FuelDeltaLiter)
	}
	e = raw(-4) // sensor error → nilai terakhir, selisih 0, bukan bacaan baru
	if ApplyFuel(prev, cal210, e) || *e.FuelLevelRaw != 1950 || *e.FuelVolumeLiter != 100 || *e.FuelDeltaLiter != 0 {
		t.Fatalf("sensor -4: raw=%v vol=%v delta=%v", *e.FuelLevelRaw, *e.FuelVolumeLiter, *e.FuelDeltaLiter)
	}
	e = raw(1755) // tanpa kalibrasi dan alat tidak mengirim liter → tidak dinilai
	if ApplyFuel(prev, nil, e) || e.FuelVolumeLiter != nil {
		t.Fatal("tanpa kalibrasi seharusnya tanpa liter")
	}
}

// data uji: menit ke-n, speed, mesin, liter
type fuelStep struct {
	min    int
	speed  float64
	engine bool
	liters float64
}

func runFuel(steps []fuelStep) []FuelOutcome {
	t0 := time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC)
	var tr Tracker
	var all []FuelOutcome
	var prev *float64
	for _, s := range steps {
		e := &Event{EventTime: t0.Add(time.Duration(s.min) * time.Minute)}
		e.SpeedKph, e.IsEngineOn = &s.speed, &s.engine
		l := s.liters
		d := 0.0 // selisih per data (seperti ApplyFuel)
		if prev != nil {
			d = l - *prev
		}
		e.FuelDeltaLiter, prev = &d, &l
		var out []FuelOutcome
		tr, out = DetectFuel(tr, e, &l, DefaultAlertRules)
		all = append(all, out...)
	}
	return all
}

func TestDetectFuel(t *testing.T) {
	t.Run("isi BBM saat berhenti: naik >= 10 L dari terendah", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, false, 50}, {1, 0, false, 49}, {2, 0, false, 70}, {3, 0, false, 120}, {4, 20, true, 120}})
		if len(out) != 2 || out[0].EventType != "increase" || out[0].DeltaLiter != 21 || out[1].DeltaLiter != 50 || out[0].AlertType != "" {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("FUD (aturan lama): turun >= 5 L sejak awal berhenti & >= 2 menit → alert, lalu diperpanjang", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, true, 100}, {1, 0, true, 94}, {2, 0, true, 93}, {3, 0, true, 85}, {4, 30, true, 85}})
		t0 := time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC)
		if len(out) != 2 || out[0].AlertType != AlertFuelDecreaseEngineOn || out[0].Extend || out[0].DeltaLiter != -7 ||
			!out[0].StartedAt.Equal(t0) || !out[1].Extend || out[1].EventType != "" {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("FUD (aturan lama): awal berhenti dihitung walau mesin sempat mati (kasus 07:26:30 replay)", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, true, 100}, {1, 0, false, 100}, {5, 0, false, 100}, {6, 0, true, 100}, {7, 0, true, 94}})
		if len(out) != 1 || out[0].AlertType != AlertFuelDecreaseEngineOn || !out[0].StartedAt.Equal(time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC)) {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("turun cepat tapi belum 2 menit lalu jalan → tanpa alert", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, true, 100}, {1, 0, true, 90}, {1, 40, true, 90}})
		if len(out) != 0 {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("turun saat bergerak tidak dinilai", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 40, true, 100}, {1, 40, true, 80}, {2, 40, true, 60}})
		if len(out) != 0 {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("turun bertahap selama mesin mati → alert saat hidup lagi (lama: tidak terdeteksi)", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, false, 100}, {10, 0, false, 97}, {20, 0, false, 94}, {30, 0, false, 92}, {31, 0, true, 92}})
		if len(out) != 1 || out[0].AlertType != AlertFuelDecreaseEngineOff || out[0].DeltaLiter != -8 {
			t.Fatalf("%+v", out)
		}
	})
	t.Run("mesin mati tanpa penurunan berarti → tanpa alert", func(t *testing.T) {
		out := runFuel([]fuelStep{{0, 0, false, 100}, {30, 0, false, 97}, {31, 0, true, 97}})
		if len(out) != 0 {
			t.Fatalf("%+v", out)
		}
	})
}
