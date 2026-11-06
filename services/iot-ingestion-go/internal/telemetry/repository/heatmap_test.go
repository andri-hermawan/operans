package repository

import (
	"math"
	"testing"
)

func TestCellDegrees(t *testing.T) {
	dLon, dLat := CellDegrees(111.32, 0)
	if math.Abs(dLat-0.001) > 1e-9 || math.Abs(dLon-0.001) > 1e-9 {
		t.Fatalf("di khatulistiwa 111,32 m = 0,001°: %v %v", dLon, dLat)
	}
	// makin jauh dari khatulistiwa, 1 derajat bujur makin pendek → ukuran derajat bujur makin besar
	if lon60, lat60 := CellDegrees(100, 60); lon60 < lat60*1.99 || lon60 > lat60*2.01 {
		t.Fatalf("pada 60° lintang dLon ≈ 2×dLat: %v %v", lon60, lat60)
	}
}
