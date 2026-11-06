package domain

import "testing"

// kalibrasi lurus unit PERCENTAGE: sensor 0..100 (%) → 0..210 L (DT00070)
var calPercent210 = &Calibration{Type: CalibrationPercentage, CapacityLiters: 210, Points: []CalPoint{{0, 0}, {100, 210}}}

func pctEvent(v float64) *Event { return &Event{FuelSensorPercent: &v} }

func prevWith(raw, liters, pct float64) *LiveState {
	p := &LiveState{}
	p.FuelLevelRaw, p.FuelVolumeLiter, p.FuelPercent = &raw, &liters, &pct
	return p
}

func TestApplyFuelPercentage(t *testing.T) {
	// bacaan normal 50%: semua kolom fuel terisi
	e := pctEvent(50)
	if !ApplyFuel(prevWith(2457, 126, 60), calPercent210, e) {
		t.Fatal("bacaan 50% seharusnya sah")
	}
	if *e.FuelLevelRaw != 2047.5 || *e.FuelVolumeLiter != 105 || *e.FuelPercent != 50 || *e.FuelDeltaLiter != -21 {
		t.Fatalf("raw=%v liter=%v persen=%v selisih=%v", *e.FuelLevelRaw, *e.FuelVolumeLiter, *e.FuelPercent, *e.FuelDeltaLiter)
	}
	if e.FuelTemperatureC != nil {
		t.Error("suhu tidak dikirim ECU → harus tetap kosong, jangan disalin dari data lama")
	}

	// data pertama (tanpa data sebelumnya): selisih 0
	e = pctEvent(100)
	if !ApplyFuel(nil, calPercent210, e) || *e.FuelLevelRaw != FuelRawScale || *e.FuelVolumeLiter != 210 || *e.FuelDeltaLiter != 0 {
		t.Fatalf("penuh: raw=%v liter=%v selisih=%v", *e.FuelLevelRaw, *e.FuelVolumeLiter, *e.FuelDeltaLiter)
	}
}

func TestApplyFuelPercentageInvalid(t *testing.T) {
	for name, v := range map[string]float64{"nol (tidak terbaca)": 0, "galat ECU 255": 255, "di atas 100": 101, "negatif": -1} {
		e := pctEvent(v)
		if ApplyFuel(prevWith(2047.5, 105, 50), calPercent210, e) {
			t.Errorf("%s: seharusnya bukan bacaan baru", name)
		}
		// kolom wajib tetap terisi dari nilai terakhir, tanpa penurunan palsu
		if *e.FuelLevelRaw != 2047.5 || *e.FuelVolumeLiter != 105 || *e.FuelPercent != 50 || *e.FuelDeltaLiter != 0 {
			t.Errorf("%s: nilai terakhir tidak dipakai", name)
		}
	}
	// IO persen tidak ada di record
	e := &Event{}
	if ApplyFuel(prevWith(2047.5, 105, 50), calPercent210, e) || e.FuelVolumeLiter == nil || *e.FuelDeltaLiter != 0 {
		t.Error("record tanpa persen: pakai nilai terakhir")
	}
	// tanpa data sebelumnya: tidak ada yang bisa disalin
	e = pctEvent(0)
	if ApplyFuel(nil, calPercent210, e) || e.FuelVolumeLiter != nil {
		t.Error("tanpa data sebelumnya, bacaan tidak valid dibiarkan kosong")
	}
}

func TestApplyFuelPercentageWithoutPoints(t *testing.T) {
	// kalibrasi hanya kapasitas (tanpa titik): dianggap lurus
	e := pctEvent(25)
	if !ApplyFuel(nil, &Calibration{Type: CalibrationPercentage, CapacityLiters: 200}, e) || *e.FuelVolumeLiter != 50 || *e.FuelPercent != 25 {
		t.Fatalf("liter=%v persen=%v", *e.FuelVolumeLiter, *e.FuelPercent)
	}
	// belum ada kalibrasi sama sekali: persen & raw setara tetap masuk, liter belum bisa dihitung
	e = pctEvent(40)
	if ApplyFuel(nil, nil, e) || e.FuelVolumeLiter != nil || *e.FuelPercent != 40 || *e.FuelLevelRaw != 1638 {
		t.Fatalf("tanpa kalibrasi: raw=%v persen=%v liter=%v", e.FuelLevelRaw, e.FuelPercent, e.FuelVolumeLiter)
	}
}

func TestApplyFuelTypeDecidesSource(t *testing.T) {
	// unit LLS tidak terpengaruh persen yang kebetulan ikut terkirim
	raw, pct := 1755.0, 10.0
	e := &Event{Fuel: Fuel{FuelLevelRaw: &raw}, FuelSensorPercent: &pct}
	if !ApplyFuel(nil, cal210, e) || *e.FuelVolumeLiter != 90 {
		t.Fatalf("LLS harus tetap dari raw: %v", e.FuelVolumeLiter)
	}
}

func TestFuelLitersOnlyPercentage(t *testing.T) {
	e := pctEvent(50)
	FuelLitersOnly(calPercent210, e)
	if *e.FuelVolumeLiter != 105 || *e.FuelPercent != 50 || *e.FuelLevelRaw != 2047.5 || e.FuelDeltaLiter != nil {
		t.Fatalf("data terlambat: liter=%v selisih=%v", *e.FuelVolumeLiter, e.FuelDeltaLiter)
	}
}
