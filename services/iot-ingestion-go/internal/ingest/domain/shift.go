package domain

import (
	"fmt"
	"time"
)

// Shift = jam kerja project (jam lokal, dari core.v_iot_shifts). End < Start = melewati tengah malam.
// Rumus sama dengan core-api (domain.Shift.At / Window) — ADR 0004.
type Shift struct {
	Code     string // SHIFT_1
	Name     string // Shift 1
	StartMin int    // menit sejak 00:00 (07:00 = 420)
	EndMin   int
	Sequence int
}

// ParseClock: "07:00" / "07:00:00" → menit sejak 00:00.
func ParseClock(s string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("jam %q tidak valid", s)
	}
	return h*60 + m, nil
}

// window = awal & akhir shift yang DIMULAI pada tanggal lokal day.
func (s Shift) window(day time.Time, loc *time.Location) (time.Time, time.Time) {
	start := time.Date(day.Year(), day.Month(), day.Day(), s.StartMin/60, s.StartMin%60, 0, 0, loc)
	endDay := day
	if s.EndMin <= s.StartMin { // melewati tengah malam (atau 24 jam)
		endDay = day.AddDate(0, 0, 1)
	}
	end := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), s.EndMin/60, s.EndMin%60, 0, 0, loc)
	return start, end
}

// ResolvePeriod menentukan tanggal operasional (YYYY-MM-DD) & shift untuk waktu t:
// shift yang memuat t (awal termasuk, akhir tidak). Data 02:00 di Shift 2 (19:00–07:00)
// milik tanggal KEMARIN (tanggal mulai shift). ok=false bila t tidak masuk shift mana pun.
func ResolvePeriod(shifts []Shift, loc *time.Location, t time.Time) (date string, shift Shift, ok bool) {
	local := t.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	for _, day := range []time.Time{today, today.AddDate(0, 0, -1)} {
		for _, s := range shifts {
			start, end := s.window(day, loc)
			if !local.Before(start) && local.Before(end) {
				return day.Format("2006-01-02"), s, true
			}
		}
	}
	return "", Shift{}, false
}
