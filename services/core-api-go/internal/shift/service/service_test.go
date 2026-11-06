package service

import (
	"testing"

	"simha/core-api/internal/shift/domain"
)

func TestShiftsOverlap(t *testing.T) {
	sh := func(start, end string) domain.Shift { return domain.Shift{Start: start, End: end} }
	for _, c := range []struct {
		a, b domain.Shift
		want bool
	}{
		{sh("07:00", "19:00"), sh("19:00", "07:00"), false}, // pola CHR: bersambung, tidak bertumpuk
		{sh("07:00", "19:00"), sh("18:00", "06:00"), true},
		{sh("19:00", "07:00"), sh("06:00", "08:00"), true}, // tumpuk setelah tengah malam
		{sh("06:00", "14:00"), sh("14:00", "22:00"), false},
	} {
		if got := ShiftsOverlap(c.a, c.b); got != c.want {
			t.Errorf("%v vs %v = %v, mau %v", c.a, c.b, got, c.want)
		}
	}
}
