package domain

import (
	"testing"
	"time"
)

func TestResolvePeriod(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	shifts := []Shift{
		{Code: "SHIFT_1", StartMin: 7 * 60, EndMin: 19 * 60, Sequence: 1},
		{Code: "SHIFT_2", StartMin: 19 * 60, EndMin: 7 * 60, Sequence: 2},
	}
	for _, c := range []struct {
		at         time.Time
		date, code string
	}{
		{time.Date(2026, 10, 22, 7, 0, 0, 0, wib), "2026-10-22", "SHIFT_1"}, // awal shift termasuk
		{time.Date(2026, 10, 22, 18, 59, 0, 0, wib), "2026-10-22", "SHIFT_1"},
		{time.Date(2026, 10, 22, 19, 0, 0, 0, wib), "2026-10-22", "SHIFT_2"}, // akhir shift 1 = awal shift 2
		{time.Date(2026, 10, 23, 2, 0, 0, 0, wib), "2026-10-22", "SHIFT_2"},  // lewat tengah malam → tanggal kemarin
		{time.Date(2026, 10, 23, 6, 59, 0, 0, wib), "2026-10-22", "SHIFT_2"},
		// jam alat dalam UTC: 22 Okt 19:30 UTC = 23 Okt 02:30 WIB → Shift 2 tanggal 22
		{time.Date(2026, 10, 22, 19, 30, 0, 0, time.UTC), "2026-10-22", "SHIFT_2"},
	} {
		date, s, ok := ResolvePeriod(shifts, wib, c.at)
		if !ok || date != c.date || s.Code != c.code {
			t.Errorf("%v → %s %s %v; mau %s %s", c.at, date, s.Code, ok, c.date, c.code)
		}
	}
	// project dengan celah jam (tanpa shift malam): di luar shift → tidak ok
	day := []Shift{{Code: "DAY", StartMin: 7 * 60, EndMin: 17 * 60}}
	if _, _, ok := ResolvePeriod(day, wib, time.Date(2026, 10, 22, 20, 0, 0, 0, wib)); ok {
		t.Error("20:00 seharusnya di luar shift DAY")
	}
}
