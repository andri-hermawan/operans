package service

import "testing"

func TestNormalizeTimeSlot(t *testing.T) {
	ok := map[string]string{"07.00-08.00": "07.00-08.00", "7:00 - 8:00": "07.00-08.00", "19.30-20.30": "19.30-20.30", "23.00-24.00": "23.00-24.00"}
	for in, want := range ok {
		if got, good := NormalizeTimeSlot(in); !good || got != want {
			t.Errorf("%q → %q,%v; want %q", in, got, good, want)
		}
	}
	for _, in := range []string{"", "07.00", "7-8", "25.00-26.00", "07.60-08.00", "abc"} {
		if got, good := NormalizeTimeSlot(in); good {
			t.Errorf("%q harus ditolak, dapat %q", in, got)
		}
	}
}
