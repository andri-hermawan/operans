package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Detektor fuel_decrease_engine_on (FUD) — keputusan pengguna 2026-11-18 (ADR 0016), menggantikan aturan lama
// "turun kumulatif sejak awal berhenti". Dasar angka: analisa 1,79 juta bacaan fuel (7 hari), goyangan sensor
// meluruh dalam ~2 menit setelah unit berhenti.
//
//	Syarat dinilai : mesin hidup, speed 0, bacaan liter sah. Speed > 0 / mesin mati / celah data > 60 dtk →
//	                 seluruh keadaan (riwayat bacaan & kejadian) di-reset.
//	Waktu tenang   : 120 dtk pertama setelah berhenti tidak dinilai (cairan tangki masih bergoyang).
//	Fluktuasi      : dalam 5 bacaan terakhir, rentang > 3 L DAN ada kenaikan ≥ 1 L (naik-turun) → jendela dihitung
//	                 ulang dari nol. Penyedotan bergerak searah turun, jadi tidak ter-reset.
//	Alert utama    : median bacaan di awal jendela − median di akhir jendela ≥ fuel_drop_liters, jendela selebar
//	                 duration_minutes (bawaan 5 L / 2 menit).
//	Penjaga lambat : setelah tenang 5 menit, turun ≥ 2 × fuel_drop_liters dalam 15 menit → alert juga, walau jendela
//	                 sempat ter-reset oleh fluktuasi (penyedotan saat sensor masih bergoyang).
//	Satu kejadian  : satu alert per rangkaian penurunan: dibuat saat syarat terpenuhi, diperpanjang selama masih
//	                 terpenuhi (akhir = bacaan terakhir yang masih turun), selisih fuel alert = TOTAL turun.
const (
	fudSettle      = 120 * time.Second // lama tenang setelah berhenti
	fudFluctLiters = 3.0               // rentang 5 bacaan yang dianggap bergoyang
	fudRiseLiters  = 1.0               // kenaikan antar-bacaan yang menandai "naik-turun"
	fudRecentN     = 5                 // jumlah bacaan terakhir untuk cek fluktuasi
	fudGap         = 60 * time.Second  // celah antar-bacaan yang me-reset keadaan
	fudEdge        = 30 * time.Second  // lebar sisi jendela untuk median awal / akhir
	fudDefaultWin  = 2 * time.Minute   // jendela bila aturan tidak punya durasi
	fudGuardSpan   = 15 * time.Minute  // rentang penjaga lambat
	fudGuardSettle = 300 * time.Second // tenang minimal sebelum penjaga lambat berlaku
	fudGuardFactor = 2.0               // ambang penjaga lambat = faktor × fuel_drop_liters
	fudKeep        = fudGuardSpan + time.Minute
)

// FuelPoint = satu bacaan liter yang sah selama rangkaian berhenti + mesin hidup.
type FuelPoint struct {
	T time.Time `json:"t"`
	L float64   `json:"l"`
}

// resetFUD menghapus seluruh keadaan FUD (unit bergerak / mesin mati / celah data).
func resetFUD(t *Tracker) {
	t.FuelBuf, t.WinFrom = nil, nil
	t.FUDOpen, t.FUDStart, t.FUDBase, t.FUDLast = false, nil, 0, nil
}

func medianOf(p []FuelPoint) float64 {
	v := make([]float64, len(p))
	for i, x := range p {
		v[i] = x.L
	}
	sort.Float64s(v)
	n := len(v)
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

// fluctuating = 5 bacaan terakhir bergoyang: rentang > fudFluctLiters dan ada kenaikan ≥ fudRiseLiters.
func fluctuating(buf []FuelPoint) bool {
	if len(buf) < fudRecentN {
		return false
	}
	w := buf[len(buf)-fudRecentN:]
	lo, hi, rise := w[0].L, w[0].L, false
	for i := 1; i < len(w); i++ {
		lo, hi = math.Min(lo, w[i].L), math.Max(hi, w[i].L)
		if w[i].L-w[i-1].L >= fudRiseLiters {
			rise = true
		}
	}
	return hi-lo > fudFluctLiters && rise
}

// detectFUD menilai satu data BERURUTAN. Dipanggil dari DetectFuel untuk setiap data; mengembalikan kejadian (alert
// baru, atau perpanjangan alert yang berjalan) atau nil. Keadaan disimpan di tracker (tahan restart layanan).
func detectFUD(t *Tracker, e *Event, liters *float64, r AlertRule, stopped, engineOn bool) *FuelOutcome {
	now := e.EventTime
	if !stopped || !engineOn {
		resetFUD(t)
		return nil
	}
	if liters == nil {
		return nil // bacaan tidak sah: waktu berjalan, tidak ada penilaian
	}
	if n := len(t.FuelBuf); n > 0 && now.Sub(t.FuelBuf[n-1].T) > fudGap {
		resetFUD(t) // celah data (sensor putus / sleep): jangan membandingkan nilai sebelum dan sesudahnya
	}
	t.FuelBuf = append(t.FuelBuf, FuelPoint{T: now, L: *liters})
	for len(t.FuelBuf) > 1 && now.Sub(t.FuelBuf[0].T) > fudKeep {
		t.FuelBuf = t.FuelBuf[1:]
	}
	if r.FuelDropLiters == nil || t.StopSince == nil {
		return nil
	}
	sinceStop := now.Sub(*t.StopSince)
	if sinceStop < fudSettle {
		return nil // masih bergoyang setelah berhenti
	}
	// fluktuasi: hanya bacaan yang semuanya sesudah waktu tenang yang dinilai
	buf := t.FuelBuf
	if len(buf) >= fudRecentN && buf[len(buf)-fudRecentN].T.Sub(*t.StopSince) >= fudSettle && fluctuating(buf) {
		w := now
		t.WinFrom = &w // jendela dihitung ulang dari bacaan berikutnya (penjaga lambat tetap jalan)
	}

	win := fudDefaultWin
	if d := minutes(r.DurationMinutes); d > 0 {
		win = d
	}
	drop, slow := 0.0, false

	// jendela utama
	var inWin []FuelPoint // bacaan sejak reset terakhir, dalam satu jendela (+ sisi) terakhir
	for _, p := range buf {
		if (t.WinFrom == nil || p.T.After(*t.WinFrom)) && now.Sub(p.T) <= win+fudEdge && p.T.Sub(*t.StopSince) >= fudSettle {
			inWin = append(inWin, p)
		}
	}
	if len(inWin) > 0 && now.Sub(inWin[0].T) >= win {
		var a, b []FuelPoint
		for _, p := range inWin {
			if !p.T.After(inWin[0].T.Add(fudEdge)) {
				a = append(a, p)
			}
			if !p.T.Before(now.Add(-fudEdge)) {
				b = append(b, p)
			}
		}
		if d := medianOf(a) - medianOf(b); d >= *r.FuelDropLiters {
			drop = d
		}
	}

	// penjaga lambat: turun besar dalam 15 menit, hanya dari bacaan yang sudah tenang
	if drop == 0 && sinceStop >= fudGuardSettle+fudGuardSpan {
		target := now.Add(-fudGuardSpan)
		var a []FuelPoint
		for _, p := range buf {
			if p.T.Sub(*t.StopSince) >= fudGuardSettle && !p.T.After(target.Add(fudEdge)) && !p.T.Before(target.Add(-fudEdge)) {
				a = append(a, p)
			}
		}
		var b []FuelPoint
		for _, p := range buf {
			if !p.T.Before(now.Add(-fudEdge)) {
				b = append(b, p)
			}
		}
		if len(a) >= 2 && len(b) >= 2 {
			if d := medianOf(a) - medianOf(b); d >= fudGuardFactor**r.FuelDropLiters {
				drop, slow = d, true
			}
		}
	}
	if drop == 0 {
		// kejadian dianggap selesai bila syarat tidak terpenuhi lagi selama satu jendela penuh
		if t.FUDOpen && t.FUDLast != nil && now.Sub(*t.FUDLast) > win {
			t.FUDOpen, t.FUDStart, t.FUDBase, t.FUDLast = false, nil, 0, nil
		}
		return nil
	}

	last := now
	if t.FUDOpen && t.FUDStart != nil && t.FUDLast != nil && now.Sub(*t.FUDLast) <= win {
		t.FUDLast = &last
		cur := medianOf(buf[max(0, len(buf)-3):])
		return &FuelOutcome{AlertType: AlertFuelDecreaseEngineOn, Extend: true, StartedAt: *t.FUDStart,
			Threshold: *r.FuelDropLiters, DeltaLiter: -round2(math.Max(drop, t.FUDBase-cur))}
	}
	start := now.Add(-win)
	if len(inWin) > 0 && !slow {
		start = inWin[0].T
	} else if slow {
		start = now.Add(-fudGuardSpan)
	}
	t.FUDOpen, t.FUDStart, t.FUDLast = true, &start, &last
	t.FUDBase = medianOf(buf[max(0, len(buf)-3):]) + drop
	desc := fmt.Sprintf("fuel turun %.2f L dalam %.0f menit saat mesin hidup & berhenti", drop, win.Minutes())
	if slow {
		desc = fmt.Sprintf("fuel turun %.2f L dalam %.0f menit saat mesin hidup & berhenti (penurunan lambat)", drop, fudGuardSpan.Minutes())
	}
	return &FuelOutcome{EventType: "decrease", AlertType: AlertFuelDecreaseEngineOn, StartedAt: start,
		Threshold: *r.FuelDropLiters, DeltaLiter: -round2(drop), Description: desc}
}
