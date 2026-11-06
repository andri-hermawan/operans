package domain

// Kode status muatan (ADR 0004).
const (
	VesselLoaded  = "loaded"
	VesselEmpty   = "empty"
	VesselUnknown = "unknown"
)

// VesselStatus = aturan status muatan SISTEM LAMA (calculators/vessel-status.ts; keputusan Fase C):
//
//	unit stop & di luar zona (fid 0)          → unknown
//	di luar zona                              → status zona terakhir di dalam (belum ada → empty)
//	ORIG_FID naik dibanding zona terakhir      → empty  (kembali kosong)
//	ORIG_FID turun                             → loaded (membawa muatan)
//	sama                                       → status sebelumnya (belum ada → empty)
//
// lastFID/lastStatus = zona TERAKHIR di dalam peta (fid ≠ 0) beserta status muatan saat itu.
func VesselStatus(fid int32, lastFID *int32, lastStatus *string, equipmentStatus string) string {
	if fid == 0 && equipmentStatus == StatusStop {
		return VesselUnknown
	}
	var previous string
	if lastStatus != nil && (*lastStatus == VesselLoaded || *lastStatus == VesselEmpty) {
		previous = *lastStatus
	}
	orEmpty := func(s string) string {
		if s == "" {
			return VesselEmpty
		}
		return s
	}
	if fid == 0 {
		return orEmpty(previous)
	}
	if lastFID != nil && *lastFID != 0 {
		switch {
		case fid > *lastFID:
			return VesselEmpty
		case fid < *lastFID:
			return VesselLoaded
		default:
			return orEmpty(previous)
		}
	}
	return VesselEmpty
}

func operationOf(e *Event) string {
	if e.EquipmentStatus != nil {
		return *e.EquipmentStatus
	}
	return ""
}

// ApplyVessel mengisi status muatan event dari kondisi terakhir unit (prev, boleh nil) dan
// memperbarui acuan "zona terakhir di dalam". Hanya untuk data BERURUTAN (bukan data terlambat).
// Tanpa posisi (zona tidak diketahui) → status & acuan dibawa dari kondisi sebelumnya; unit yang belum punya
// kondisi sebelumnya dianggap di luar zona (fid 0) sehingga hasilnya tetap terisi (empty / unknown), seperti sistem lama.
func ApplyVessel(prev *LiveState, e *Event) {
	var lastFID *int32
	var lastStatus, prevVessel *string
	if prev != nil {
		lastFID, lastStatus, prevVessel = prev.LastInsideFID, prev.LastInsideVesselStatus, prev.VesselStatus
	}
	if e.GeofenceFID == nil {
		e.VesselStatus, e.LastInsideFID, e.LastInsideVesselStatus = prevVessel, lastFID, lastStatus
		if e.VesselStatus == nil {
			status := VesselStatus(0, lastFID, lastStatus, operationOf(e))
			e.VesselStatus = &status
		}
		return
	}
	op := operationOf(e)
	fid := *e.GeofenceFID
	status := VesselStatus(fid, lastFID, lastStatus, op)
	e.VesselStatus = &status
	if fid != 0 {
		e.LastInsideFID, e.LastInsideVesselStatus = &fid, &status
	} else {
		e.LastInsideFID, e.LastInsideVesselStatus = lastFID, lastStatus
	}
}

// ApplyVesselLate mengisi status muatan untuk data TERLAMBAT (alat sempat offline lalu mengirim ulang data lama).
// Urutan data ini tidak diketahui, jadi acuan "zona terakhir di dalam" dipakai apa adanya dan TIDAK diperbarui
// (kondisi terakhir unit tidak boleh terganggu data lama). Hasilnya perkiraan, tetapi kolom tidak pernah kosong.
func ApplyVesselLate(prev *LiveState, e *Event) {
	var lastFID *int32
	var lastStatus, prevVessel *string
	if prev != nil {
		lastFID, lastStatus, prevVessel = prev.LastInsideFID, prev.LastInsideVesselStatus, prev.VesselStatus
	}
	if e.GeofenceFID == nil && prevVessel != nil { // tanpa posisi: ikut status terakhir unit
		e.VesselStatus = prevVessel
		return
	}
	var fid int32
	if e.GeofenceFID != nil {
		fid = *e.GeofenceFID
	}
	status := VesselStatus(fid, lastFID, lastStatus, operationOf(e))
	e.VesselStatus = &status
}
