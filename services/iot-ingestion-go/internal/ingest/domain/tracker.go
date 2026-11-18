package domain

import (
	"fmt"
	"time"
)

// RefuelLiters = kenaikan minimum saat unit berhenti yang dicatat sebagai isi BBM (ADR 0005/0009;
// sistem lama tidak punya aturan ini).
const RefuelLiters = 10.0

// Tracker = keadaan pendeteksi kejadian satu unit; disimpan di live state (kolom tracker, JSON) agar
// kejadian yang sedang berjalan tidak hilang saat layanan restart.
type Tracker struct {
	// Unit berhenti (speed 0, mesin apa pun) — rangkaian yang SAMA dengan sistem lama (findStopStreakStart):
	// dipakai isi BBM & fuel_decrease_engine_on
	StopSince      *time.Time `json:"stop_since,omitempty"`
	StopStartLiter *float64   `json:"stop_start_liter,omitempty"` // liter awal berhenti (tidak lagi dipakai FUD sejak ADR 0016)
	StopMinLiter   *float64   `json:"stop_min_liter,omitempty"`   // liter terendah selama berhenti (isi BBM)

	// fuel_decrease_engine_on (ADR 0016): riwayat bacaan selama berhenti + mesin hidup, awal jendela setelah
	// fluktuasi, dan kejadian penurunan yang sedang berjalan
	FuelBuf  []FuelPoint `json:"fuel_buf,omitempty"`
	WinFrom  *time.Time  `json:"win_from,omitempty"`  // jendela dihitung dari bacaan SESUDAH waktu ini
	FUDOpen  bool        `json:"fud_open,omitempty"`  // alert FUD kejadian ini sudah dibuat
	FUDStart *time.Time  `json:"fud_start,omitempty"` // awal alert (kunci perpanjangan)
	FUDBase  float64     `json:"fud_base,omitempty"`  // liter patokan awal kejadian (untuk total turun)
	FUDLast  *time.Time  `json:"fud_last,omitempty"`  // bacaan terakhir yang masih memenuhi syarat

	// Mesin mati: fuel_decrease_engine_off
	OffSince      *time.Time `json:"off_since,omitempty"`
	OffStartLiter *float64   `json:"off_start_liter,omitempty"` // liter saat mesin mati (bacaan pertama)

	// Kejadian area terakhir (aturan geofence lama): enter / exit + nama area
	GeoLastEvent   string `json:"geo_last_event,omitempty"`
	GeoLastSegment string `json:"geo_last_segment,omitempty"`

	// Alert berdurasi (speed, off-track)
	Spans map[string]Span `json:"spans,omitempty"` // kunci = jenis alert
}

// Span = satu kondisi alert yang sedang berlangsung.
type Span struct {
	Since time.Time `json:"since"`
	Last  time.Time `json:"last"`           // data terakhir yang masih memenuhi kondisi
	Open  bool      `json:"open,omitempty"` // alert sudah dibuat (durasi tercapai)
}

// FuelOutcome = kejadian fuel yang terdeteksi pada satu data.
type FuelOutcome struct {
	EventType   string    // increase | decrease ("" = hanya perpanjang alert, tanpa kejadian fuel baru)
	DeltaLiter  float64   // negatif = turun (kumulatif periode)
	StartedAt   time.Time // awal periode (berhenti / mesin mati)
	AlertType   string    // "" = tanpa alert (isi BBM)
	Extend      bool      // true = perpanjang ended_at alert yang sudah ada (StartedAt sama)
	Threshold   float64   // ambang aturan saat itu
	Description string
}

// DetectFuel menilai satu data BERURUTAN. liters = bacaan liter baru yang sah (nil = tidak ada / tidak valid:
// waktu periode tetap berjalan, tetapi tidak ada penilaian). Aturan:
//   - isi BBM (konsep baru, ADR 0009): selama berhenti, naik ≥ 10 L dari liter terendah → increase
//   - fuel_decrease_engine_off (konsep baru): liter saat mesin mati − liter saat hidup lagi ≥ fuel_drop_liters
//   - fuel_decrease_engine_on (ADR 0016, menggantikan aturan lama 2026-10-26): median turun ≥ fuel_drop_liters
//     dalam jendela duration_minutes saat mesin hidup & berhenti, dengan waktu tenang dan reset bila sensor
//     bergoyang — rincian di fuel_decrease.go. Data yang sudah memicu fuel_decrease_engine_off tidak dinilai FUD.
//   - naik/turun saat bergerak tidak dinilai
func DetectFuel(t Tracker, e *Event, liters *float64, rules AlertRules) (Tracker, []FuelOutcome) {
	now := e.EventTime
	stopped := num(e.SpeedKph) == 0
	engineOn := e.IsEngineOn != nil && *e.IsEngineOn
	var out []FuelOutcome

	// rangkaian berhenti + isi BBM
	if stopped {
		if t.StopSince == nil {
			t.StopSince, t.StopStartLiter, t.StopMinLiter = &now, nil, nil
			resetFUD(&t)
		}
		if liters != nil {
			if t.StopStartLiter == nil {
				t.StopStartLiter = copyF(liters)
			}
			switch {
			case t.StopMinLiter == nil || *liters < *t.StopMinLiter:
				t.StopMinLiter = copyF(liters)
			case *liters-*t.StopMinLiter >= RefuelLiters:
				d := round2(*liters - *t.StopMinLiter)
				out = append(out, FuelOutcome{EventType: "increase", DeltaLiter: d, StartedAt: *t.StopSince,
					Threshold: RefuelLiters, Description: fmt.Sprintf("isi BBM %.2f L", d)})
				t.StopMinLiter = copyF(liters)
			}
		}
	} else {
		t.StopSince, t.StopStartLiter, t.StopMinLiter = nil, nil, nil
		resetFUD(&t)
	}

	// fuel turun selama mesin mati (dinilai saat mesin hidup lagi)
	fuo := false
	if !engineOn {
		if t.OffSince == nil {
			t.OffSince, t.OffStartLiter = &now, copyF(liters)
		} else if t.OffStartLiter == nil {
			t.OffStartLiter = copyF(liters)
		}
	} else if t.OffSince != nil && liters != nil { // tanpa bacaan sah: tunggu bacaan berikutnya
		r := rules[AlertFuelDecreaseEngineOff]
		if t.OffStartLiter != nil && r.FuelDropLiters != nil {
			drop := *t.OffStartLiter - *liters
			if drop > 0 && drop >= *r.FuelDropLiters {
				out = append(out, FuelOutcome{EventType: "decrease", DeltaLiter: -round2(drop), StartedAt: *t.OffSince,
					AlertType: AlertFuelDecreaseEngineOff, Threshold: *r.FuelDropLiters,
					Description: fmt.Sprintf("fuel turun %.2f L selama mesin mati", drop)})
				fuo = true
			}
		}
		t.OffSince, t.OffStartLiter = nil, nil
	}

	// fuel turun saat mesin hidup & berhenti (ADR 0016, lihat fuel_decrease.go). Data yang sudah memicu
	// fuel_decrease_engine_off tidak dinilai FUD (urutan lama), tetapi riwayat bacaannya tetap dicatat.
	if fo := detectFUD(&t, e, liters, rules[AlertFuelDecreaseEngineOn], stopped, engineOn); fo != nil && !fuo {
		out = append(out, *fo)
	}
	return t, out
}

func minutes(p *float64) time.Duration { return time.Duration(num(p) * float64(time.Minute)) }

func copyF(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
