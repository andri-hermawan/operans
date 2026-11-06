package domain

import (
	"testing"
	"time"
)

func TestShift(t *testing.T) {
	shifts := []Shift{
		{Code: "SHIFT_1", Name: "Shift 1", Start: "07:00", End: "19:00", Sequence: 1},
		{Code: "SHIFT_2", Name: "Shift 2", Start: "19:00", End: "07:00", Sequence: 2},
	}
	for _, in := range []string{"Shift 2", "SHIFT_2", "shift2", "2", " SHIFT 2 "} {
		if s, ok := ResolveShift(shifts, in); !ok || s.Code != "SHIFT_2" {
			t.Errorf("%q tidak dikenali sebagai SHIFT_2", in)
		}
	}
	if _, ok := ResolveShift(shifts, "Shift 3"); ok {
		t.Error("Shift 3 seharusnya tidak dikenal")
	}

	wib, _ := time.LoadLocation("Asia/Jakarta")
	at, _ := shifts[1].At("2026-10-22", 2, 0, wib) // Shift 2, jam 02:00 → hari berikutnya
	if at.Format("2006-01-02 15:04") != "2026-10-23 02:00" {
		t.Fatalf("Shift 2 02:00 → %s", at)
	}
	at, _ = shifts[1].At("2026-10-22", 21, 0, wib)
	if at.Format("2006-01-02 15:04") != "2026-10-22 21:00" {
		t.Fatalf("Shift 2 21:00 → %s", at)
	}
}
