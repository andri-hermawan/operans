package domain

import (
	"reflect"
	"testing"
)

func TestKeysFor(t *testing.T) {
	for path, want := range map[string][]string{
		"/api/v1/weighbridge-tickets":             {"upload/weighbridge"},
		"/api/v1/weighbridge-tickets/123":         {"upload/weighbridge"},
		"/api/v1/rain-slippery-logs":              {"upload/rain-slippery"},
		"/api/v1/imports/ritases/template":        {"upload/ritase"},
		"/api/v1/fuel-fillings":                   {"upload/fuel-filling"},
		"/api/v1/imports/fuel-fillings":           {"upload/fuel-filling"},
		"/api/v1/imports/work-hour-notifications": {"upload/wh-notification"},
		"/api/v1/reports/executive-hauling":       {"report/report-executive-hauling"},
		"/api/v1/reports/executive-emplacement":   {"report/report-executive-emplacement"},
		"/api/v1/shifts":                          nil, // tidak dijaga
		"/api/v1/equipment":                       nil,
		"/api/v1/access/mine":                     nil,
		"/api/v1/imports/tidak-ada":               nil,
		"/api/v1/imports":                         nil,
		"/health":                                 nil,
	} {
		if got := KeysFor(path); !reflect.DeepEqual(got, want) {
			t.Errorf("KeysFor(%q) = %v, mau %v", path, got, want)
		}
	}
	// rekap harian juga dipakai Dashboard, jadi "dashboard" ikut sebagai salah satu halaman
	if k := KeysFor("/api/v1/reports/weighbridge-daily"); len(k) < 2 || k[0] != "dashboard" {
		t.Errorf("weighbridge-daily harus menyertakan dashboard: %v", k)
	}
}

func TestDefaultAllowed(t *testing.T) {
	for _, c := range []struct {
		role, key string
		want      bool
	}{
		{"viewer", "upload/weighbridge", false},
		{"viewer", "report/report-ritase-daily", false},
		{"viewer", "dashboard", true},
		{"dispatcher", "upload/weighbridge", true},
		{"dispatcher", "report/report-ritase-daily", true},
		{"admin", "report/report-ritase-daily", true},
		{"admin", "report/equipment-shift-performance", true},
		{"dispatcher", "report/equipment-shift-performance", false},
	} {
		if got := DefaultAllowed(c.role, c.key); got != c.want {
			t.Errorf("DefaultAllowed(%s, %s) = %v, mau %v", c.role, c.key, got, c.want)
		}
	}
}

func TestResolve(t *testing.T) {
	none := map[string]bool{}
	// bawaan viewer tertutup; aturan role membukakan
	if !Resolve("viewer", "report/report-breakdown-daily", map[string]bool{"report/report-breakdown-daily": true}, none) {
		t.Error("aturan role harus membukakan Report untuk viewer")
	}
	// aturan user menimpa role
	if Resolve("viewer", "report/report-breakdown-daily", map[string]bool{"report/report-breakdown-daily": true}, map[string]bool{"report/report-breakdown-daily": false}) {
		t.Error("aturan user (blokir) harus menimpa role (izinkan)")
	}
	// dispatcher diblokir
	if Resolve("dispatcher", "upload/ritase", map[string]bool{"upload/ritase": false}, none) {
		t.Error("blokir harus berlaku")
	}
	// halaman khusus admin tidak bisa dibukakan
	if Resolve("dispatcher", "report/equipment-shift-performance", map[string]bool{"report/equipment-shift-performance": true}, none) {
		t.Error("halaman khusus admin tidak boleh dibukakan")
	}
}
