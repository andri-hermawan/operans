package domain

import (
	"math"
	"sort"
)

// Jenis kalibrasi tangki (core.fuel_calibrations.calibration_type, ADR 0008).
const (
	CalibrationLLS        = "lls"        // nilai mentah sensor level (mis. Escort 0–4095)
	CalibrationPercentage = "percentage" // persen dari ECU (0–100)
)

// FuelRawScale = skala nilai mentah sensor LLS (0–4095). Unit bertipe percentage menyimpan "raw setara" pada skala
// yang sama (persen × 4095 ÷ 100), supaya kolom fuel_level_raw selalu terisi, searah dengan isi tangki, dan liter
// masih bisa dihitung ulang bila kalibrasi diperbaiki.
const FuelRawScale = 4095.0

// Calibration = tabel kalibrasi tangki satu equipment (dari core.v_iot_fuel_calibration_points, ADR 0008).
type Calibration struct {
	Type           string // lls | percentage
	CapacityLiters float64
	Points         []CalPoint // urut nilai sensor naik
}

type CalPoint struct{ Sensor, Liters float64 }

// Liters = interpolasi linear (rumus sama dengan core-api domain.Liters). ok=false bila di luar rentang.
func (c *Calibration) Liters(sensor float64) (liters, percent float64, ok bool) {
	p := c.Points
	n := len(p)
	if n == 0 || math.IsNaN(sensor) || sensor < p[0].Sensor || sensor > p[n-1].Sensor {
		return 0, 0, false
	}
	i := sort.Search(n, func(i int) bool { return p[i].Sensor >= sensor })
	if p[i].Sensor == sensor || i == 0 {
		liters = p[i].Liters
	} else {
		lo, hi := p[i-1], p[i]
		liters = lo.Liters + (sensor-lo.Sensor)*(hi.Liters-lo.Liters)/(hi.Sensor-lo.Sensor)
	}
	if c.CapacityLiters > 0 {
		percent = liters / c.CapacityLiters * 100
	}
	return round2(liters), round2(percent), true
}

// InvalidFuelReading: sensor LLS mengirim −4 (error) atau 0 (tidak terbaca) — aturan lama.
func InvalidFuelReading(raw float64) bool { return raw == 0 || raw == -4 }

// InvalidFuelPercent: persen dari ECU yang tidak bisa dipercaya — kosong/0 (tidak terbaca; truk yang berjalan tidak
// mungkin 0%) atau di atas 100 (mis. 255 = kode galat ECU). Sama sikapnya dengan InvalidFuelReading pada LLS.
func InvalidFuelPercent(pct float64) bool { return math.IsNaN(pct) || pct <= 0 || pct > 100 }

// isPercentage = unit ini memakai sensor persen (ECU): dari kalibrasinya, atau — bila belum ada kalibrasi — dari
// kenyataan bahwa alat hanya mengirim persen.
func isPercentage(cal *Calibration, e *Event) bool {
	if cal != nil {
		return cal.Type == CalibrationPercentage
	}
	return e.FuelSensorPercent != nil && e.FuelLevelRaw == nil
}

// percentFuel menghitung seluruh nilai fuel dari persen ECU: raw setara, liter, dan persen terkalibrasi.
// liters=nil bila liter tidak bisa dihitung (tanpa kalibrasi maupun kapasitas).
func percentFuel(cal *Calibration, pct float64) (raw float64, liters, percent *float64) {
	raw = round2(pct * FuelRawScale / 100)
	percent = &pct
	if cal == nil {
		return raw, nil, percent
	}
	if l, p, ok := cal.Liters(pct); ok { // titik kalibrasi (tangki tidak lurus pun benar)
		return raw, &l, &p
	}
	if cal.CapacityLiters > 0 { // tanpa titik: anggap lurus
		l := round2(pct / 100 * cal.CapacityLiters)
		return raw, &l, percent
	}
	return raw, nil, percent
}

// applyPercentFuel = ApplyFuel untuk unit bertipe percentage (sensor ECU, AVL ID 89 / FUEL_PERCENT_IO_ID).
// Semua kolom fuel diisi: raw setara, liter, persen, selisih. Suhu dibiarkan apa adanya (ECU umumnya tidak mengirim).
// Bacaan kosong / tidak valid → nilai terakhir dipakai, selisih 0 (supaya kolom tetap terisi dan tidak ada
// penurunan palsu), dan hasilnya bukan bacaan baru.
func applyPercentFuel(prev *LiveState, cal *Calibration, e *Event) (fuelValid bool) {
	if e.FuelSensorPercent == nil || InvalidFuelPercent(*e.FuelSensorPercent) {
		if prev != nil && prev.FuelLevelRaw != nil {
			zero := 0.0
			e.FuelLevelRaw, e.FuelVolumeLiter, e.FuelPercent = prev.FuelLevelRaw, prev.FuelVolumeLiter, prev.FuelPercent
			e.FuelDeltaLiter = &zero
		}
		return false
	}
	raw, liters, pct := percentFuel(cal, *e.FuelSensorPercent)
	e.FuelLevelRaw, e.FuelVolumeLiter, e.FuelPercent = &raw, liters, pct
	if liters == nil {
		return false
	}
	delta := 0.0
	if prev != nil && prev.FuelVolumeLiter != nil {
		delta = round2(*liters - *prev.FuelVolumeLiter)
	}
	e.FuelDeltaLiter = &delta
	return true
}

// ApplyFuel mengisi liter, persen, dan selisih liter event (data BERURUTAN):
//   - unit bertipe percentage (sensor ECU) → applyPercentFuel
//   - bacaan tidak valid (0 / −4) → nilai fuel terakhir dipakai, selisih 0
//     (lama: selisih terakhir ikut disalin → penurunan terhitung dua kali)
//   - ada kalibrasi → liter & persen dihitung dari nilai mentah; selisih = liter sekarang − liter terakhir
//   - tanpa kalibrasi → nilai liter dari alat (bila ada) dibiarkan
//
// fuelValid=false bila liter event ini bukan bacaan baru (tidak dipakai pendeteksi kejadian).
func ApplyFuel(prev *LiveState, cal *Calibration, e *Event) (fuelValid bool) {
	if isPercentage(cal, e) {
		return applyPercentFuel(prev, cal, e)
	}
	if e.FuelLevelRaw == nil {
		return false
	}
	if InvalidFuelReading(*e.FuelLevelRaw) {
		if prev != nil && prev.FuelLevelRaw != nil {
			zero := 0.0
			e.FuelLevelRaw, e.FuelVolumeLiter, e.FuelPercent = prev.FuelLevelRaw, prev.FuelVolumeLiter, prev.FuelPercent
			e.FuelTemperatureC, e.FuelDeltaLiter = prev.FuelTemperatureC, &zero
		}
		return false
	}
	if cal != nil {
		if l, pct, ok := cal.Liters(*e.FuelLevelRaw); ok {
			e.FuelVolumeLiter, e.FuelPercent = &l, &pct
		} else {
			e.FuelVolumeLiter, e.FuelPercent = nil, nil // di luar rentang kalibrasi: tidak dihitung (sama dengan lama)
		}
	}
	if e.FuelVolumeLiter == nil {
		return false
	}
	delta := 0.0
	if prev != nil && prev.FuelVolumeLiter != nil {
		delta = round2(*e.FuelVolumeLiter - *prev.FuelVolumeLiter)
	}
	e.FuelDeltaLiter = &delta
	return true
}

// FuelLitersOnly = liter & persen untuk data TERLAMBAT (tanpa selisih: urutannya tidak diketahui).
func FuelLitersOnly(cal *Calibration, e *Event) {
	if isPercentage(cal, e) {
		if e.FuelSensorPercent != nil && !InvalidFuelPercent(*e.FuelSensorPercent) {
			raw, liters, pct := percentFuel(cal, *e.FuelSensorPercent)
			e.FuelLevelRaw, e.FuelVolumeLiter, e.FuelPercent = &raw, liters, pct
		}
		e.FuelDeltaLiter = nil
		return
	}
	if cal == nil || e.FuelLevelRaw == nil || InvalidFuelReading(*e.FuelLevelRaw) {
		return
	}
	if l, pct, ok := cal.Liters(*e.FuelLevelRaw); ok {
		e.FuelVolumeLiter, e.FuelPercent = &l, &pct
	}
	e.FuelDeltaLiter = nil
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
