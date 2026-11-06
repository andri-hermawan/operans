package dailyinput

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/excel"
	shiftdomain "simha/core-api/internal/shift/domain"
)

// ---------------------------------------------------------------------
// Bantuan bersama untuk input harian (memeriksa satu isian; kesalahan dikumpulkan di errs)
// ---------------------------------------------------------------------

// ApplyDateShift: tanggal (YYYY-MM-DD / DD-MM-YYYY / sel Excel) + shift. ok=true bila keduanya valid.
func (pd *ProjectData) ApplyDateShift(date *string, dateIn, shiftIn *string, errs *[]RowIssue) (shiftdomain.Shift, bool) {
	ok := false
	if v, has := Required(dateIn, "operational_date", errs); has {
		if d, err := excel.ParseDate(v); err != nil {
			*errs = append(*errs, RowIssue{Column: "operational_date", Message: err.Error()})
		} else {
			*date, ok = d, true
		}
	}
	var sh shiftdomain.Shift
	if v, has := Required(shiftIn, "shift", errs); has {
		var found bool
		if sh, found = shiftdomain.ResolveShift(pd.Shifts, v); !found {
			*errs = append(*errs, RowIssue{Column: "shift", Message: "shift \"" + v + "\" tidak dikenal di project ini"})
			ok = false
		}
	} else {
		ok = false
	}
	return sh, ok
}

// ClockAt mengubah jam (HH:MM / sel jam Excel) menjadi waktu lengkap menurut shift. Kosong → nil.
func (pd *ProjectData) ClockAt(sh shiftdomain.Shift, date string, in *string, col string, errs *[]RowIssue) *time.Time {
	if in == nil {
		return nil
	}
	hh, mm, has, err := excel.ParseClock(*in)
	if err != nil {
		*errs = append(*errs, RowIssue{Column: col, Message: col + ": " + err.Error()})
		return nil
	}
	if !has {
		return nil
	}
	t, err := sh.At(date, hh, mm, pd.Loc)
	if err != nil {
		*errs = append(*errs, RowIssue{Column: col, Message: err.Error()})
		return nil
	}
	return &t
}

// ClockInShift = ClockAt + wajib di dalam jam shift (batas awal & akhir termasuk).
func (pd *ProjectData) ClockInShift(sh shiftdomain.Shift, date string, in *string, col string, from, to time.Time, errs *[]RowIssue) *time.Time {
	t := pd.ClockAt(sh, date, in, col, errs)
	if t != nil && (t.Before(from) || t.After(to)) {
		*errs = append(*errs, RowIssue{Column: col,
			Message: fmt.Sprintf("%s %s di luar jam %s (%s–%s)", col, t.Format("15:04"), sh.Name, sh.Start, sh.End)})
		return nil
	}
	return t
}

// RefCode mencocokkan label/kode ke daftar pilihan; hasil = kode baku.
func (pd *ProjectData) RefCode(group, text, col string, errs *[]RowIssue) string {
	if v, ok := pd.Refs[group][NormRef(text)]; ok {
		return v.Code
	}
	*errs = append(*errs, RowIssue{Column: col, Message: col + " \"" + text + "\" tidak ada di daftar pilihan"})
	return ""
}

// SetEquipment: kode unit huruf besar + tautan ke master bila ada (ADR 0007: master tidak wajib).
func (pd *ProjectData) SetEquipment(code *string, id **uuid.UUID, raw string, errs *[]RowIssue, col string) {
	c := strings.ToUpper(strings.TrimSpace(raw))
	if utf8.RuneCountInString(c) > 50 {
		*errs = append(*errs, RowIssue{Column: col, Message: col + " maks. 50 karakter"})
		return
	}
	*code = c
	*id = nil
	if eid, ok := pd.Equipment[c]; ok {
		*id = &eid
	}
}

func NormRef(s string) string {
	return strings.NewReplacer(" ", "", "_", "", "-", "").Replace(strings.ToLower(strings.TrimSpace(s)))
}

// Required: isian wajib (nil atau kosong → kesalahan).
func Required(p *string, col string, errs *[]RowIssue) (string, bool) {
	if p == nil || strings.TrimSpace(*p) == "" {
		*errs = append(*errs, RowIssue{Column: col, Message: col + " wajib diisi"})
		return "", false
	}
	return strings.TrimSpace(*p), true
}

// Optional: teks boleh kosong (→ NULL), dengan batas panjang.
func Optional(p *string, col string, max int, errs *[]RowIssue) *string {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	v := strings.TrimSpace(*p)
	if utf8.RuneCountInString(v) > max {
		*errs = append(*errs, RowIssue{Column: col, Message: fmt.Sprintf("%s maks. %d karakter", col, max)})
	}
	return &v
}

// Overlay menyalin setiap field pointer patch yang tidak nil ke dst (struct isian yang sama).
func Overlay[T any](dst *T, patch T) {
	d, p := reflect.ValueOf(dst).Elem(), reflect.ValueOf(patch)
	for i := range p.NumField() {
		if f := p.Field(i); f.Kind() == reflect.Pointer && !f.IsNil() {
			d.Field(i).Set(f)
		}
	}
}

func ClockOf(t *time.Time, loc *time.Location) *string {
	if t == nil {
		return nil
	}
	s := t.In(loc).Format("15:04")
	return &s
}

// HoursText: 1.5 → "01:30".
func HoursText(h float64) string {
	m := int(math.Round(h * 60))
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}
