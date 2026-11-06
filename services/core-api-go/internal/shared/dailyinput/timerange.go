package dailyinput

import (
	"fmt"
	"math"
	"strings"
	"time"

	"simha/core-api/internal/shared/validate"
	shiftdomain "simha/core-api/internal/shift/domain"
)

// RangeCols = nama isian untuk pesan kesalahan sebuah pasangan jam mulai/selesai + total jam.
type RangeCols struct{ Start, End, Hours string }

// Range memeriksa SATU pasangan jam mulai/selesai beserta total jamnya (aturan sama dengan Status Breakdown):
//   - jam mulai dan selesai diisi bersamaan atau dikosongkan bersamaan
//   - jam selesai tidak boleh lebih awal dari jam mulai; shift lewat tengah malam → jam setelah 00:00 = hari berikutnya
//   - total jam kosong → dihitung dari jam; diisi tetapi beda lebih dari 0,02 jam dari selisih jam → ditolak
//   - tanpa jam, total jam yang diisi manual dipakai (0–24 jam)
//
// Jam tidak wajib berada di dalam jam shift (kejadian bisa melewati pergantian shift). dateOK=false → hanya total jam diperiksa.
func (pd *ProjectData) Range(sh shiftdomain.Shift, date string, dateOK bool, startIn, endIn *string, hoursIn *float64,
	c RangeCols, errs *[]RowIssue) (start, end *time.Time, hours *float64) {

	if hoursIn != nil && !math.IsNaN(*hoursIn) { // NaN = sudah dilaporkan pengurai Excel
		if h := *hoursIn; h < 0 || h > 24 {
			*errs = append(*errs, RowIssue{Column: c.Hours, Message: c.Hours + " harus 0–24 jam"})
		} else {
			h = validate.Round2(h)
			hours = &h
		}
	}
	if !dateOK {
		return nil, nil, hours
	}
	filled := func(p *string) bool { return p != nil && strings.TrimSpace(*p) != "" }
	if filled(startIn) != filled(endIn) {
		*errs = append(*errs, RowIssue{Column: c.Start, Message: c.Start + " dan " + c.End + " harus diisi bersamaan"})
		return nil, nil, hours
	}
	start = pd.ClockAt(sh, date, startIn, c.Start, errs)
	end = pd.ClockAt(sh, date, endIn, c.End, errs)
	if start == nil || end == nil {
		return nil, nil, hours // kosong, atau salah format (sudah dilaporkan)
	}
	if end.Before(*start) {
		*errs = append(*errs, RowIssue{Column: c.End, Message: c.End + " lebih awal dari " + c.Start})
		return nil, nil, hours
	}
	d := validate.Round2(end.Sub(*start).Hours())
	if hours != nil && math.Abs(*hours-d) > 0.02 {
		*errs = append(*errs, RowIssue{Column: c.Hours,
			Message: fmt.Sprintf("%s %s tidak sama dengan %s − %s (%s)", c.Hours, HoursText(*hours), c.End, c.Start, HoursText(d))})
	}
	return start, end, &d
}
