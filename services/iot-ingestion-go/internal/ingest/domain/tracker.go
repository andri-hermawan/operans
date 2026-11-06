package domain

import (
	"fmt"
	"math"
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
	StopStartLiter *float64   `json:"stop_start_liter,omitempty"` // liter awal berhenti (acuan FUD)
	StopMinLiter   *float64   `json:"stop_min_liter,omitempty"`   // liter terendah selama berhenti (isi BBM)
	FUDOpen        bool       `json:"fud_open,omitempty"`         // alert FUD rangkaian berhenti ini sudah dibuat

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
//   - fuel_decrease_engine_on = SISTEM LAMA (keputusan 2026-10-26): data ini mesin hidup, berhenti, dan liter
//     turun; turun kumulatif sejak AWAL BERHENTI (speed 0, mesin hidup/mati) ≥ fuel_drop_liters (dibulatkan
//     1 desimal seperti lama) dan lama berhenti ≥ duration_minutes → alert (awal = awal berhenti, akhir = data
//     ini); data berikutnya yang masih memenuhi → akhir alert diperpanjang. Data yang sudah memicu
//     fuel_decrease_engine_off tidak dinilai FUD (urutan lama).
//   - naik/turun saat bergerak tidak dinilai
func DetectFuel(t Tracker, e *Event, liters *float64, rules AlertRules) (Tracker, []FuelOutcome) {
	now := e.EventTime
	stopped := num(e.SpeedKph) == 0
	engineOn := e.IsEngineOn != nil && *e.IsEngineOn
	var out []FuelOutcome

	// rangkaian berhenti + isi BBM
	if stopped {
		if t.StopSince == nil {
			t.StopSince, t.StopStartLiter, t.StopMinLiter, t.FUDOpen = &now, nil, nil, false
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
		t.StopSince, t.StopStartLiter, t.StopMinLiter, t.FUDOpen = nil, nil, nil, false
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

	// fuel turun saat mesin hidup & berhenti (aturan lama)
	r := rules[AlertFuelDecreaseEngineOn]
	if !fuo && engineOn && stopped && liters != nil && t.StopStartLiter != nil && r.FuelDropLiters != nil &&
		e.FuelDeltaLiter != nil && *e.FuelDeltaLiter < 0 {
		cum := *liters - *t.StopStartLiter
		if math.Round(cum*10)/10 <= -*r.FuelDropLiters && now.Sub(*t.StopSince) >= minutes(r.DurationMinutes) {
			o := FuelOutcome{StartedAt: *t.StopSince, AlertType: AlertFuelDecreaseEngineOn, Threshold: *r.FuelDropLiters,
				DeltaLiter: round2(cum)}
			if t.FUDOpen {
				o.Extend = true
			} else {
				o.EventType = "decrease"
				o.Description = fmt.Sprintf("fuel turun %.2f L saat mesin hidup & berhenti", -cum)
				t.FUDOpen = true
			}
			out = append(out, o)
		}
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
