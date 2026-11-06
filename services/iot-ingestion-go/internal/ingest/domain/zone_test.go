package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func zone(t *testing.T, name string, priority int, area float64, geojson string) Zone {
	t.Helper()
	z := Zone{ID: uuid.New(), Name: name, Priority: priority, AreaM2: area, ValidFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	if err := z.SetGeoJSON(geojson); err != nil {
		t.Fatal(err)
	}
	return z
}

func TestZoneContains(t *testing.T) {
	// kotak 0..10 dengan lubang 4..6
	z := zone(t, "Kotak", 0, 100, `{"type":"MultiPolygon","coordinates":[[[[0,0],[10,0],[10,10],[0,10],[0,0]],[[4,4],[6,4],[6,6],[4,6],[4,4]]]]}`)
	for _, c := range []struct {
		x, y float64
		in   bool
	}{
		{1, 1, true}, {5, 5, false}, {10, 5, true}, {0, 0, true}, {11, 5, false}, {-0.0001, 5, false},
	} {
		if got := z.Contains(c.x, c.y); got != c.in {
			t.Errorf("Contains(%v,%v) = %v", c.x, c.y, got)
		}
	}
}

func TestFindZone(t *testing.T) {
	now := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	besar := zone(t, "Trase", 0, 30000, `{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10],[0,0]]]}`)
	kecil := zone(t, "CSA", 0, 1800, `{"type":"Polygon","coordinates":[[[2,2],[4,2],[4,4],[2,4],[2,2]]]}`)
	prio := zone(t, "Jembatan", 5, 25000, `{"type":"Polygon","coordinates":[[[0,0],[5,0],[5,5],[0,5],[0,0]]]}`)
	lama := zone(t, "Versi lama", 9, 1, `{"type":"Polygon","coordinates":[[[0,0],[10,0],[10,10],[0,10],[0,0]]]}`)
	akhir := now.Add(-time.Hour)
	lama.ValidTo = &akhir

	if z := FindZone([]Zone{besar, kecil}, 3, 3, now); z == nil || z.Name != "CSA" {
		t.Fatalf("bertumpuk → terkecil: %+v", z)
	}
	if z := FindZone([]Zone{besar, kecil, prio, lama}, 3, 3, now); z == nil || z.Name != "Jembatan" {
		t.Fatalf("priority menang, versi lama diabaikan: %+v", z)
	}
	if z := FindZone([]Zone{besar, lama}, 8, 8, now.Add(-2*time.Hour)); z == nil || z.Name != "Versi lama" {
		t.Fatalf("data lampau memakai peta yang berlaku saat itu: %+v", z)
	}
	if z := FindZone([]Zone{besar}, 20, 20, now); z != nil {
		t.Fatal("di luar semua zona harus nil")
	}
}

func TestVesselStatusAturanLama(t *testing.T) {
	i := func(v int32) *int32 { return &v }
	s := func(v string) *string { return &v }
	for _, c := range []struct {
		fid        int32
		lastFID    *int32
		lastStatus *string
		op, want   string
	}{
		{0, i(5), s(VesselLoaded), StatusStop, VesselUnknown},   // stop di luar zona
		{0, i(5), s(VesselLoaded), StatusRunning, VesselLoaded}, // di luar zona: status dipertahankan
		{0, nil, nil, StatusRunning, VesselEmpty},               // tanpa riwayat
		{6, i(5), s(VesselLoaded), StatusRunning, VesselEmpty},  // fid naik
		{4, i(5), s(VesselEmpty), StatusRunning, VesselLoaded},  // fid turun
		{5, i(5), s(VesselLoaded), StatusIdle, VesselLoaded},    // sama
		{5, i(5), s(VesselUnknown), StatusIdle, VesselEmpty},    // sama, status lama tidak dikenal
		{5, nil, nil, StatusRunning, VesselEmpty},               // zona pertama
		{4, i(5), s(VesselEmpty), StatusStop, VesselLoaded},     // stop DI DALAM zona tetap dihitung
	} {
		if got := VesselStatus(c.fid, c.lastFID, c.lastStatus, c.op); got != c.want {
			t.Errorf("VesselStatus(%d,%v,%v,%s) = %s; mau %s", c.fid, c.lastFID, c.lastStatus, c.op, got, c.want)
		}
	}
}

// Aturan geofence lama: kejadian hanya untuk area ber-ORIG_FID 0 (rute seperti simulator DT03:
// CSA → trase → Jembatan Timbang → Km 01 → Km 02 → luar jalur).
func TestGeofenceTransitionsAturanLama(t *testing.T) {
	type step struct {
		fid    int32
		inside bool
		name   string
	}
	var tr Tracker
	var prev *LiveState
	var log []string
	for _, s := range []step{
		{0, true, "CSA Simpang Sidomulyo"},   // fid 0 & belum ada kejadian → exit CSA (sama dengan lama)
		{1, true, "Trase Simpang Sidomulyo"}, // dari fid 0 ke fid ≠ 0, kejadian terakhir exit → tidak ada
		{0, true, "Jembatan Timbang Km 01"},  // area fid 0 lain, terakhir exit → enter
		{0, true, "Jembatan Timbang Km 01"},  // area sama → tidak ada
		{2, true, "Km 01+000 - 02+000"},      // keluar dari fid 0 ke jalan, terakhir enter → exit Jembatan Timbang
		{3, true, "Km 02+000 - 03+000"},      // antar segmen jalan → tidak ada
		{0, false, ""},                       // luar jalur → tidak dinilai
	} {
		fid, inside, name := s.fid, s.inside, s.name
		e := &Event{}
		e.GeofenceFID, e.IsInsideGeofence = &fid, &inside
		if inside {
			e.SegmentName = &name
		}
		for _, g := range GeofenceTransitions(prev, e, &tr) {
			log = append(log, g.EventType+":"+deref(g.SegmentName))
		}
		prev = &LiveState{GeofenceFID: &fid, IsInsideGeofence: &inside, SegmentName: e.SegmentName}
	}
	want := "exit:CSA Simpang Sidomulyo,enter:Jembatan Timbang Km 01,exit:Jembatan Timbang Km 01"
	if got := strings.Join(log, ","); got != want {
		t.Fatalf("dapat %q\nmau   %q", got, want)
	}
}
