package domain

import (
	"time"

	"github.com/google/uuid"
)

// Jenis kejadian zona (ADR 0004).
const (
	GeofenceEnter = "enter"
	GeofenceExit  = "exit"
)

// GeofenceEvent = unit masuk / keluar zona (tabel iot.geofence_events; lama: geofences IN/OUT).
type GeofenceEvent struct {
	ID               int64      `json:"id"`
	ProjectID        *uuid.UUID `json:"project_id"`
	EquipmentID      uuid.UUID  `json:"equipment_id"`
	DeviceID         uuid.UUID  `json:"device_id"`
	TelemetryEventID int64      `json:"telemetry_event_id"`
	EventType        string     `json:"event_type"` // enter | exit
	EventTime        time.Time  `json:"event_time"`
	OperationalDate  *string    `json:"operational_date"`
	ShiftCode        *string    `json:"shift_code"`

	// Zona yang dimasuki / ditinggalkan
	ZoneID           *uuid.UUID `json:"zone_id"`
	GeofenceFID      *int32     `json:"geofence_fid"`
	LocationCategory *string    `json:"location_category"`
	SegmentName      *string    `json:"segment_name"`

	// Kondisi saat kejadian
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	SpeedKph        *float64 `json:"speed_kph"`
	IsEngineOn      *bool    `json:"is_engine_on"`
	EquipmentStatus *string  `json:"equipment_status"`
	VesselStatus    *string  `json:"vessel_status"`
	FuelLevelRaw    *float64 `json:"fuel_level_raw"`
	MileageM        *float64 `json:"mileage_m"`
}

// GeofenceTransitions = kejadian masuk/keluar area ATURAN SISTEM LAMA (geofence-event.service, keputusan
// 2026-10-26): kejadian hanya untuk AREA KHUSUS ber-ORIG_FID 0 (CSA, Jembatan Timbang, Emplacement, …),
// bukan setiap pindah segmen jalan. t menyimpan kejadian terakhir (lama: query tabel geofences per paket).
//
//	exit  : (di area ber-fid 0 & belum pernah ada kejadian)  atau
//	        (data sebelumnya fid 0, sekarang fid ≠ 0, kejadian terakhir = enter) → area yang DITINGGALKAN
//	enter : sekarang di area ber-fid 0, kejadian terakhir = exit, dan namanya beda dengan area kejadian terakhir
//
// Hanya bila zona data ini diketahui DAN di dalam zona (lama: segment ≠ "Unknown").
// Data lengkap (waktu, posisi, shift, …) diisi pemanggil dari event yang tersimpan.
func GeofenceTransitions(prev *LiveState, e *Event, t *Tracker) []GeofenceEvent {
	if e.GeofenceFID == nil || e.IsInsideGeofence == nil || !*e.IsInsideGeofence {
		return nil
	}
	var prevFID int32
	if prev != nil && prev.GeofenceFID != nil {
		prevFID = *prev.GeofenceFID
	}
	cur := *e.GeofenceFID
	segment := deref(e.SegmentName)
	var out []GeofenceEvent
	if (cur == 0 && t.GeoLastEvent == "") || (prevFID == 0 && cur != 0 && t.GeoLastEvent == GeofenceEnter) {
		g := GeofenceEvent{EventType: GeofenceExit, ZoneID: e.ZoneID, GeofenceFID: e.GeofenceFID,
			LocationCategory: e.LocationCategory, SegmentName: e.SegmentName}
		if prev != nil { // area yang ditinggalkan = data sebelumnya (lama: lastLog)
			g.ZoneID, g.GeofenceFID, g.LocationCategory, g.SegmentName = prev.ZoneID, prev.GeofenceFID, prev.LocationCategory, prev.SegmentName
		}
		out = append(out, g)
		t.GeoLastEvent, t.GeoLastSegment = GeofenceExit, deref(g.SegmentName)
	}
	if cur == 0 && t.GeoLastEvent == GeofenceExit && t.GeoLastSegment != segment {
		out = append(out, GeofenceEvent{EventType: GeofenceEnter, ZoneID: e.ZoneID, GeofenceFID: e.GeofenceFID,
			LocationCategory: e.LocationCategory, SegmentName: e.SegmentName})
		t.GeoLastEvent, t.GeoLastSegment = GeofenceEnter, segment
	}
	return out
}

// deref = isi teks, "" bila nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
