package domain

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// langkah uji: detik ke-n, speed, di dalam zona (nil = zona tidak diketahui)
type step struct {
	sec    int
	speed  float64
	inside *bool
}

func yes() *bool { v := true; return &v }
func no() *bool  { v := false; return &v }

// runAlerts mengembalikan ringkasan "detik:aksi:jenis:awal-akhir" (detik sejak t0) agar mudah dibandingkan.
func runAlerts(rules AlertRules, steps []step) string {
	t0 := time.Date(2026, 10, 23, 8, 0, 0, 0, time.UTC)
	sec := func(x time.Time) int { return int(x.Sub(t0).Seconds()) }
	var tr Tracker
	var log []string
	for _, s := range steps {
		e := &Event{EventTime: t0.Add(time.Duration(s.sec) * time.Second)}
		sp := s.speed
		e.SpeedKph, e.IsInsideGeofence = &sp, s.inside
		var out []AlertOutcome
		tr, out = DetectAlerts(tr, e, rules)
		for _, o := range out {
			end := "terbuka"
			if o.EndedAt != nil {
				end = fmt.Sprint(sec(*o.EndedAt))
			}
			log = append(log, fmt.Sprintf("%d:%s:%s:%d-%s", s.sec, o.Action, o.AlertType, sec(o.StartedAt), end))
		}
	}
	return strings.Join(log, ",")
}

// Akhir alert = aturan sistem lama (keputusan 2026-10-26).
func TestDetectAlerts(t *testing.T) {
	r := DefaultAlertRules // overspeed 50 km/j 1 menit, underspeed 10 km/j 2 menit, off_track 3 menit
	for _, c := range []struct {
		name  string
		steps []step
		want  string
	}{
		{"overspeed: dibuat & ditutup saat terdeteksi (1 menit), satu alert per rangkaian",
			[]step{{0, 60, yes()}, {30, 65, yes()}, {60, 70, yes()}, {90, 55, yes()}, {120, 40, yes()}, {150, 60, yes()}},
			"60:open:overspeed:0-60"},
		{"overspeed singkat (< 1 menit) tidak menjadi alert",
			[]step{{0, 60, yes()}, {30, 40, yes()}, {60, 60, yes()}, {90, 45, yes()}}, ""},
		{"underspeed: dibuat saat 2 menit, diperpanjang selama masih underspeed",
			[]step{{0, 5, yes()}, {60, 6, yes()}, {120, 7, yes()}, {150, 8, yes()}, {180, 0, yes()}},
			"120:open:underspeed:0-120,150:extend:underspeed:0-150"},
		{"off_track: akhir = jam terakhir berhenti di luar zona; jalan di luar tidak menutup",
			[]step{{0, 0, no()}, {120, 0, no()}, {180, 0, no()}, {240, 20, no()}, {300, 0, no()}, {360, 20, yes()}},
			"180:open:off_track:0-terbuka,360:close:off_track:0-300"},
		{"off_track: berhenti di luar putus oleh jalan sebelum 3 menit → hitungan ulang",
			[]step{{0, 0, no()}, {120, 20, no()}, {150, 0, no()}, {300, 0, no()}, {330, 0, no()}},
			"330:open:off_track:150-terbuka"},
		{"zona tidak diketahui (tanpa peta/posisi) → off_track tidak dinilai",
			[]step{{0, 0, nil}, {600, 0, nil}}, ""},
	} {
		if got := runAlerts(r, c.steps); got != c.want {
			t.Errorf("%s:\n  dapat %q\n  mau   %q", c.name, got, c.want)
		}
	}

	// batas speed per zona (aturan efektif): 30 km/j, durasi 0
	zona := AlertRules{AlertOverspeed: {DurationMinutes: f(0), SpeedLimitKph: f(30)}}
	if got := runAlerts(zona, []step{{0, 35, yes()}, {30, 25, yes()}}); got != "0:open:overspeed:0-0" {
		t.Errorf("batas zona 30 km/j, durasi 0: %q", got)
	}
}
