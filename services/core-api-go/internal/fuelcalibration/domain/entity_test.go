package domain

import (
	"math"
	"testing"
)

func TestLinearPointsSamaDenganLama(t *testing.T) {
	// data fms_lite (unit 209 L, LLS 4095): 5 L = 98, 70 L = 1372, 75 L = 1469, 205 L = 4017, lalu 209 L = 4095
	p := LinearPoints(209, 4095, CalibrationLLS)
	if len(p) != 43 || p[1].SensorValue != 98 || p[14].SensorValue != 1372 || p[15].SensorValue != 1469 ||
		p[41] != (CalibrationPoint{4017, 205}) || p[42] != (CalibrationPoint{4095, 209}) {
		t.Fatalf("209 L: n=%d %v %v %v %v", len(p), p[1], p[14], p[15], p[42])
	}
	// kapasitas kelipatan 5 → titik terakhir sudah = kapasitas (tidak dobel)
	p = LinearPoints(210, 4095, CalibrationLLS)
	if len(p) != 43 || p[42] != (CalibrationPoint{4095, 210}) {
		t.Fatalf("210 L: n=%d last=%v", len(p), p[len(p)-1])
	}
	// ECU persen: 2 desimal
	p = LinearPoints(300, 100, CalibrationPercentage)
	if p[1].SensorValue != 1.67 || p[len(p)-1] != (CalibrationPoint{100, 300}) {
		t.Fatalf("persen: %v %v", p[1], p[len(p)-1])
	}
}

func TestLiters(t *testing.T) {
	p := []CalibrationPoint{{0, 0}, {100, 10}, {300, 50}, {400, 50}}
	for _, c := range []struct {
		v    float64
		want float64
		ok   bool
	}{
		{0, 0, true}, {50, 5, true}, {100, 10, true}, {200, 30, true}, {350, 50, true}, {400, 50, true},
		{-1, 0, false}, {401, 0, false},
	} {
		got, ok := Liters(p, c.v)
		if ok != c.ok || math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Liters(%v) = %v,%v; mau %v,%v", c.v, got, ok, c.want, c.ok)
		}
	}
	if _, ok := Liters(nil, 1); ok {
		t.Error("tanpa titik seharusnya tidak dihitung")
	}
}
