package masterdata

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// unitKey = project + kode unit (status breakdown dipegang per unit, bukan per shift).
type unitKey struct {
	project uuid.UUID
	code    string
}

// newerBreakdown = a lebih baru dari b menurut URUTAN INPUT: waktu terakhir disimpan/diubah/diunggah (updated_at);
// bila sama (satu unggahan menyimpan banyak baris sekaligus) → tanggal operasional, shift, lalu jam mulai.
func newerBreakdown(a, b BreakdownRow) bool {
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	if a.OperationalDate != b.OperationalDate {
		return a.OperationalDate > b.OperationalDate
	}
	if a.ShiftCode != b.ShiftCode {
		return a.ShiftCode > b.ShiftCode
	}
	return startOf(&a).After(startOf(&b))
}

func startOf(r *BreakdownRow) time.Time {
	if r.StartedAt == nil {
		return time.Time{}
	}
	return *r.StartedAt
}

// breakdownsByUnit mengelompokkan catatan breakdown per unit, yang terbaru diinput di depan.
func breakdownsByUnit(byDay map[dayKey][]BreakdownRow) map[unitKey][]BreakdownRow {
	out := map[unitKey][]BreakdownRow{}
	for k, rows := range byDay {
		uk := unitKey{k.project, k.equipmentCode}
		out[uk] = append(out[uk], rows...)
	}
	for _, rows := range out {
		sort.Slice(rows, func(i, j int) bool { return newerBreakdown(rows[i], rows[j]) })
	}
	return out
}

// stickyBreakdown = keadaan breakdown sebuah unit menurut catatan TERBARU yang diinput (rows: terbaru di depan).
// Aturan (menempel): status breakdown / continue → breakdown, DIBIARKAN sampai ada input / ubah / unggah status ready
// (jam mulai/selesai, pergantian shift, dan unit yang ternyata bergerak tidak membatalkannya; tim yang memantau
// breakdown yang mengubah statusnya). Deskripsi (mis. "AC HOT") hanya ada selama masih breakdown: dari catatan terbaru,
// atau — bila kosong — dari catatan sebelumnya dalam rangkaian breakdown yang sama (berhenti di ready).
// known=false bila unit belum punya catatan: pemanggil membiarkan nilai yang sedang berlaku.
func stickyBreakdown(rows []BreakdownRow) (isBreakdown bool, description *string, known bool) {
	if len(rows) == 0 {
		return false, nil, false
	}
	if rows[0].Status == "ready" {
		return false, nil, true
	}
	for _, r := range rows {
		if r.Status == "ready" {
			break
		}
		if r.Description != nil && strings.TrimSpace(*r.Description) != "" {
			return true, r.Description, true
		}
	}
	return true, nil, true
}

// Operational = keadaan operasional sebuah unit menurut data induk.
type Operational struct {
	BreakdownKnown       bool // false = unit belum punya catatan breakdown → pertahankan nilai yang ada
	IsBreakdown          bool
	BreakdownDescription *string
	OperatorKnown        bool // false = shift pada waktu itu belum diketahui → pertahankan operator yang ada
	Operator             *string
}

// OperationalAt = keadaan unit pada waktu t: status breakdown TERBARU yang diinput (menempel, tidak bergantung jam) dan
// operator pada shift yang berlaku di waktu t. Dipakai memperbarui kondisi terakhir unit SEGERA setelah breakdown /
// operator diinput, diubah, diunggah, atau dihapus — tanpa menunggu telemetry berikutnya (unit offline pun ikut).
// ok=false bila project/unit belum diketahui.
func (c *Cache) OperationalAt(projectID, equipmentID *uuid.UUID, t time.Time) (op Operational, ok bool) {
	if projectID == nil || equipmentID == nil {
		return op, false
	}
	s := c.snap.Load()
	code, found := s.equipment[*equipmentID]
	if !found {
		return op, false
	}
	op.IsBreakdown, op.BreakdownDescription, op.BreakdownKnown = stickyBreakdown(s.unitBreakdowns[unitKey{*projectID, code}])
	if p, found := s.projects[*projectID]; found {
		if date, shift, found := domain.ResolvePeriod(p.shifts, p.loc, t); found {
			op.OperatorKnown = true
			if name, has := s.operators[dayKey{*projectID, date, shift.Code, code}]; has {
				op.Operator = &name
			}
		}
	}
	return op, true
}
