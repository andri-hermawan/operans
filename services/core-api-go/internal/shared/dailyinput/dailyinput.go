// Package dailyinput = bahan bersama untuk INPUT HARIAN (operator, breakdown, weighbridge, dan impor Excel):
// memuat data project (zona waktu, shift, master equipment, daftar pilihan) sekali, lalu dipakai untuk
// memeriksa setiap isian/baris Excel. Data dimuat LEWAT SERVICE fitur pemiliknya (project, shift,
// equipment, reference) — interface-nya ada di file ini dan disambungkan di cmd/api/main.go.
package dailyinput

import (
	"context"
	"time"

	"github.com/google/uuid"

	projectdomain "simha/core-api/internal/project/domain"
	referencedomain "simha/core-api/internal/reference/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
	shiftdomain "simha/core-api/internal/shift/domain"
)

// RowIssue = satu kesalahan/peringatan pada baris file (nomor baris = nomor di Excel).
type RowIssue struct {
	Row     int    `json:"row,omitempty"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

// DailyFilter = filter umum data harian. Tanggal YYYY-MM-DD; kosong = tidak difilter.
type DailyFilter struct {
	ProjectID     *uuid.UUID
	DateFrom      string
	DateTo        string
	ShiftID       *uuid.UUID
	EquipmentCode string
}

// CheckFilter: project sesuai hak lihat pemanggil + format tanggal.
func CheckFilter(a access.Actor, f *DailyFilter) error {
	scope, err := access.ViewScope(a, f.ProjectID)
	if err != nil {
		return err
	}
	f.ProjectID = scope
	for _, d := range []string{f.DateFrom, f.DateTo} {
		if d != "" && !validate.IsDate(d) {
			return apperror.Invalid("tanggal harus format YYYY-MM-DD")
		}
	}
	return nil
}

// ---------------------------------------------------------------------
// Sumber data (dipenuhi service fitur lain; disambungkan di main.go)
// ---------------------------------------------------------------------

// ProjectFinder = fitur project: project aktif berdasarkan id (nil bila tidak ada / nonaktif).
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

// ShiftLister = fitur shift: shift AKTIF project, urut sequence.
type ShiftLister interface {
	ListActive(ctx context.Context, projectID uuid.UUID) ([]shiftdomain.Shift, error)
}

// EquipmentCodes = fitur equipment: kode unit → id semua equipment project.
type EquipmentCodes interface {
	CodeMap(ctx context.Context, projectID uuid.UUID) (map[string]uuid.UUID, error)
}

// ReferenceLister = fitur reference: daftar pilihan aktif (group kosong = semua kelompok).
type ReferenceLister interface {
	List(ctx context.Context, group string) ([]referencedomain.ReferenceValue, error)
}

// Loader memuat ProjectData dari keempat sumber di atas.
type Loader struct {
	Projects   ProjectFinder
	Shifts     ShiftLister
	Equipment  EquipmentCodes
	References ReferenceLister
}

// ProjectData = data project yang dibutuhkan untuk memvalidasi input harian.
type ProjectData struct {
	Project   *projectdomain.Project
	Loc       *time.Location
	Shifts    []shiftdomain.Shift
	Equipment map[string]uuid.UUID
	Refs      map[string]map[string]referencedomain.ReferenceValue // kelompok → label/kode (dinormalkan) → pilihan
}

// Load menentukan project tujuan (access.TargetProject) & memuat shift + master equipment-nya.
//
// write=true → butuh hak isi (admin/dispatcher); false → cukup hak lihat.
func (l Loader) Load(ctx context.Context, a access.Actor, requested *uuid.UUID, write bool) (*ProjectData, error) {
	pid, err := access.TargetProject(a, requested)
	if err != nil {
		return nil, err
	}
	if (write && !a.CanOperate(pid)) || (!write && !a.CanView(pid)) {
		return nil, apperror.ErrForbidden
	}
	p, err := l.Projects.FindActive(ctx, pid)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return nil, err
	}
	shifts, err := l.Shifts.ListActive(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	eq, err := l.Equipment.CodeMap(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	refList, err := l.References.List(ctx, "")
	if err != nil {
		return nil, err
	}
	refs := map[string]map[string]referencedomain.ReferenceValue{}
	for _, v := range refList {
		if refs[v.GroupCode] == nil {
			refs[v.GroupCode] = map[string]referencedomain.ReferenceValue{}
		}
		refs[v.GroupCode][NormRef(v.Label)] = v
		refs[v.GroupCode][NormRef(v.Code)] = v
	}
	return &ProjectData{Project: p, Loc: loc, Shifts: shifts, Equipment: eq, Refs: refs}, nil
}
