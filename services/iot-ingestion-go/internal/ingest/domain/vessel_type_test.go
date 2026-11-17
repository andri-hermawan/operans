package domain

import "testing"

func vesselEvent(typ string, inside *bool, fid int32) *Event {
	e := &Event{}
	if typ != "" {
		e.EquipmentType = &typ
	}
	e.IsInsideGeofence = inside
	e.GeofenceFID = &fid
	return e
}

func boolp(b bool) *bool { return &b }

// Hanya Dump Truck yang punya loaded/empty. Water/Fuel Truck: di dalam zona kosong, di luar zona (off-track) unknown.
func TestApplyVesselNonDump(t *testing.T) {
	cases := []struct {
		name   string
		typ    string
		inside *bool
		want   string // "" = nil
	}{
		{"water di dalam zona → kosong", "Water Truck", boolp(true), ""},
		{"fuel di luar zona → unknown", "Fuel Truck", boolp(false), VesselUnknown},
		{"fuel tanpa posisi → kosong", "Fuel Truck", nil, ""},
		{"dump di dalam zona → empty (aturan lama)", "Dump Truck", boolp(true), VesselEmpty},
		{"tanpa tipe → aturan dump truck", "", boolp(true), VesselEmpty},
	}
	for _, c := range cases {
		fid := int32(5)
		if c.inside != nil && !*c.inside {
			fid = 0
		}
		e := vesselEvent(c.typ, c.inside, fid)
		ApplyVessel(nil, e)
		got := ""
		if e.VesselStatus != nil {
			got = *e.VesselStatus
		}
		if got != c.want {
			t.Errorf("%s: dapat %q, harap %q", c.name, got, c.want)
		}
		if c.typ == "Water Truck" || c.typ == "Fuel Truck" {
			if e.LastInsideFID != nil || e.LastInsideVesselStatus != nil {
				t.Errorf("%s: acuan zona terakhir harus kosong", c.name)
			}
		}
	}
}

func TestApplyVesselLateNonDump(t *testing.T) {
	e := vesselEvent("Water Truck", boolp(false), 0)
	ApplyVesselLate(nil, e)
	if e.VesselStatus == nil || *e.VesselStatus != VesselUnknown {
		t.Fatalf("data terlambat unit non-DT di luar zona harus unknown, dapat %v", e.VesselStatus)
	}
}
