package database

import "github.com/google/uuid"

// Filter standar halaman pemantauan (Fase D/F3). Semua memakai tanggal operasional + kode shift (ADR 0004),
// dan selalu dibatasi project pemanggil (ProjectID nil = superadmin, semua project).
// Diisi dari query URL oleh response.ReadSlot / response.ReadRange, lalu dipakai repository fitur
// telemetry, geofence, alert, dan fuel.

// Slot = tanggal operasional (wajib) + shift (opsional) + equipment (opsional).
type Slot struct {
	ProjectID   *uuid.UUID
	Date        string // YYYY-MM-DD
	ShiftCode   *string
	EquipmentID *uuid.UUID
}

// SlotWhere = potongan WHERE untuk Slot; parameternya $1..$4 = Slot.Args().
const SlotWhere = `operational_date = $1::date AND ($2::text IS NULL OR shift_code = $2)
   AND ($3::uuid IS NULL OR project_id = $3) AND ($4::uuid IS NULL OR equipment_id = $4)`

// Args = nilai parameter $1..$4 untuk SlotWhere.
func (s Slot) Args() []any { return []any{s.Date, s.ShiftCode, s.ProjectID, s.EquipmentID} }

// RangeFilter: rentang tanggal operasional [From, To] + shift/jenis/unit opsional.
type RangeFilter struct {
	ProjectID   *uuid.UUID
	From, To    string
	ShiftCode   *string
	Types       []string
	EquipmentID *uuid.UUID
	Limit       int
}
