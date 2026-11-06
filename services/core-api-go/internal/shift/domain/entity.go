// Package domain (fitur shift) = jam kerja project: bentuk data Shift dan aturan murninya
// (mencocokkan teks "Shift 1", menghitung jam pada shift yang melewati tengah malam).
package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Shift = jam kerja project (jam lokal). End < Start berarti melewati tengah malam (mis. 19:00–07:00).
type Shift struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Code      string    `json:"shift_code"` // SHIFT_1
	Name      string    `json:"shift_name"` // Shift 1
	Start     string    `json:"start_time"` // HH:MM
	End       string    `json:"end_time"`   // HH:MM
	Sequence  int       `json:"sequence"`
}

// ResolveShift mencocokkan teks dari form/Excel ke shift project.
// Diterima: nama ("Shift 1"), kode ("SHIFT_1"), atau urutan ("1"); tidak peka huruf besar/spasi.
func ResolveShift(shifts []Shift, text string) (Shift, bool) {
	norm := func(s string) string {
		return strings.NewReplacer(" ", "", "_", "", "-", "").Replace(strings.ToUpper(strings.TrimSpace(s)))
	}
	t := norm(text)
	if t == "" {
		return Shift{}, false
	}
	n, isNum := strconv.Atoi(t)
	for _, s := range shifts {
		if norm(s.Name) == t || norm(s.Code) == t || (isNum == nil && s.Sequence == n) {
			return s, true
		}
	}
	return Shift{}, false
}

// At mengubah jam HH:MM pada tanggal operasional menjadi waktu lengkap di zona waktu project.
// Pada shift yang melewati tengah malam, jam sebelum jam mulai shift jatuh pada HARI BERIKUTNYA
// (ADR 0004): Shift 2 tanggal 22 Okt, jam 02:00 → 23 Okt 02:00.
func (s Shift) At(operationalDate string, hh, mm int, loc *time.Location) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", operationalDate, loc)
	if err != nil {
		return time.Time{}, err
	}
	startMin, err := clockMinutes(s.Start)
	if err != nil {
		return time.Time{}, err
	}
	endMin, err := clockMinutes(s.End)
	if err != nil {
		return time.Time{}, err
	}
	at := hh*60 + mm
	if endMin < startMin && at < startMin { // melewati tengah malam, bagian setelah 00:00
		day = day.AddDate(0, 0, 1)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc), nil
}

func clockMinutes(hhmm string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return 0, fmt.Errorf("jam shift %q tidak valid", hhmm)
	}
	return h*60 + m, nil
}

// Window = awal & akhir shift pada tanggal operasional (akhir hari berikutnya bila melewati tengah malam).
func (s Shift) Window(operationalDate string, loc *time.Location) (start, end time.Time, err error) {
	sm, err := clockMinutes(s.Start)
	if err != nil {
		return
	}
	em, err := clockMinutes(s.End)
	if err != nil {
		return
	}
	if start, err = s.At(operationalDate, sm/60, sm%60, loc); err != nil {
		return
	}
	if end, err = s.At(operationalDate, em/60, em%60, loc); err != nil {
		return
	}
	if !end.After(start) { // shift 24 jam (mulai = selesai)
		end = end.AddDate(0, 0, 1)
	}
	return start, end, nil
}

// ShiftRow = shift termasuk yang nonaktif (halaman kelola shift).
type ShiftRow struct {
	Shift
	IsActive  bool      `json:"is_active"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ShiftInput struct {
	ProjectID *uuid.UUID `json:"project_id"` // saat tambah; admin = project aktifnya
	Code      *string    `json:"shift_code"`
	Name      *string    `json:"shift_name"`
	Start     *string    `json:"start_time"` // HH:MM
	End       *string    `json:"end_time"`
	Sequence  *int       `json:"sequence"`
	IsActive  *bool      `json:"is_active"`
}
