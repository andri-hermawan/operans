package domain

import "time"

// Aksi alert berdurasi.
const (
	AlertOpen   = "open"   // buat alert (EndedAt kosong = masih terbuka)
	AlertExtend = "extend" // perpanjang ended_at alert yang sama (StartedAt sama)
	AlertClose  = "close"  // tutup alert terbuka (ended_at = EndedAt)
)

// AlertOutcome = alert yang harus dibuat / diperpanjang / ditutup karena satu data.
type AlertOutcome struct {
	Action    string
	AlertType string
	StartedAt time.Time  // awal kondisi
	EndedAt   *time.Time // akhir alert (open: kosong = terbuka)
	Threshold *float64   // open: batas speed / durasi (off_track)
}

// DetectAlerts menilai alert berdurasi untuk satu data BERURUTAN. Awal alert = awal kondisi bila kondisi
// bertahan ≥ duration_minutes. AKHIR alert MENGIKUTI SISTEM LAMA (keputusan pengguna 2026-10-26, hasil
// pembandingan replay fms_lite):
//
//	overspeed  : speed > batas            → dibuat SEKALIGUS ditutup saat terdeteksi (akhir = jam terdeteksi);
//	                                        satu alert per rangkaian ngebut
//	underspeed : 0 < speed < batas        → dibuat saat terdeteksi; akhir diperpanjang ke setiap data yang masih
//	                                        underspeed (akhir = data underspeed terakhir)
//	off_track  : di luar zona & speed 0   → dibuat terbuka; ditutup saat kembali DI DALAM zona dengan
//	                                        akhir = jam TERAKHIR unit berhenti di luar zona; jalan di luar zona
//	                                        memutus hitungan durasi, tetapi alert yang sudah terbuka tetap terbuka
//
// Zona tidak diketahui (tanpa posisi / peta) → off_track tidak dinilai (keadaan dipertahankan).
func DetectAlerts(t Tracker, e *Event, rules AlertRules) (Tracker, []AlertOutcome) {
	spans := map[string]Span{}
	for k, v := range t.Spans {
		spans[k] = v
	}
	now := e.EventTime
	var out []AlertOutcome
	at := func(x time.Time) *time.Time { return &x }

	// mulai / lanjutkan rangkaian kondisi; ok=true bila durasi tercapai pada data ini atau sebelumnya
	begin := func(typ string) (Span, bool) {
		sp, has := spans[typ]
		if !has {
			sp = Span{Since: now}
		}
		sp.Last = now
		return sp, now.Sub(sp.Since) >= minutes(rules[typ].DurationMinutes)
	}

	speed := num(e.SpeedKph)

	// overspeed
	if limit := rules[AlertOverspeed].SpeedLimitKph; limit != nil && speed > *limit {
		sp, due := begin(AlertOverspeed)
		if due && !sp.Open {
			sp.Open = true
			out = append(out, AlertOutcome{Action: AlertOpen, AlertType: AlertOverspeed, StartedAt: sp.Since, EndedAt: at(now), Threshold: limit})
		}
		spans[AlertOverspeed] = sp
	} else {
		delete(spans, AlertOverspeed)
	}

	// underspeed
	if limit := rules[AlertUnderspeed].SpeedLimitKph; limit != nil && speed > 0 && speed < *limit {
		sp, due := begin(AlertUnderspeed)
		switch {
		case due && !sp.Open:
			sp.Open = true
			out = append(out, AlertOutcome{Action: AlertOpen, AlertType: AlertUnderspeed, StartedAt: sp.Since, EndedAt: at(now), Threshold: limit})
		case sp.Open:
			out = append(out, AlertOutcome{Action: AlertExtend, AlertType: AlertUnderspeed, StartedAt: sp.Since, EndedAt: at(now)})
		}
		spans[AlertUnderspeed] = sp
	} else {
		delete(spans, AlertUnderspeed)
	}

	// off_track
	if e.IsInsideGeofence != nil {
		inside := *e.IsInsideGeofence
		sp, has := spans[AlertOffTrack]
		switch {
		case !inside && speed == 0:
			var due bool
			sp, due = begin(AlertOffTrack)
			if due && !sp.Open {
				sp.Open = true
				out = append(out, AlertOutcome{Action: AlertOpen, AlertType: AlertOffTrack, StartedAt: sp.Since,
					Threshold: rules[AlertOffTrack].DurationMinutes})
			}
			spans[AlertOffTrack] = sp
		case inside && has && sp.Open:
			out = append(out, AlertOutcome{Action: AlertClose, AlertType: AlertOffTrack, StartedAt: sp.Since, EndedAt: at(sp.Last)})
			delete(spans, AlertOffTrack)
		case has && !sp.Open:
			delete(spans, AlertOffTrack) // kondisi putus sebelum durasi tercapai
		}
	}

	t.Spans = spans
	if len(t.Spans) == 0 {
		t.Spans = nil
	}
	return t, out
}
