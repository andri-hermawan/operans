// Package plan = bahan bersama fitur rencana produksi (yearlyplan, monthlyplan): filter daftar, isian teks/tahun/bulan,
// dan deret angka per bulan atau per tanggal yang di JSON ditampilkan sebagai kolom datar (jan..dec / day_01..day_31).
package plan

import (
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/validate"
)

// Filter = filter daftar rencana. Year/Month 0 = tidak difilter; Partner kosong = semua mitra.
type Filter struct {
	ProjectID *uuid.UUID
	Year      int
	Month     int
	Partner   string
}

// CheckFilter: project sesuai hak lihat pemanggil + tahun/bulan masuk akal.
func CheckFilter(a access.Actor, f *Filter) error {
	scope, err := access.ViewScope(a, f.ProjectID)
	if err != nil {
		return err
	}
	f.ProjectID = scope
	if f.Year != 0 && (f.Year < MinYear || f.Year > MaxYear) {
		return apperror.Invalid("year harus %d sampai %d", MinYear, MaxYear)
	}
	if f.Month != 0 && (f.Month < 1 || f.Month > 12) {
		return apperror.Invalid("month harus 1 sampai 12")
	}
	return nil
}

const (
	MinYear = 2000
	MaxYear = 2100
	maxAmt  = 1e12 // numeric(16,2) menampung sampai 1e14; batas ini cukup untuk target produksi
)

// ---------------------------------------------------------------------
// JSON datar
// ---------------------------------------------------------------------

// MarshalFlat membuat objek JSON dari isian tetap + deret angka (kolom cols, nilainya vals).
func MarshalFlat(fixed map[string]any, cols []string, vals []float64) ([]byte, error) {
	m := make(map[string]any, len(fixed)+len(cols))
	for k, v := range fixed {
		m[k] = v
	}
	for i, c := range cols {
		m[c] = vals[i]
	}
	return json.Marshal(m)
}

// SplitInput membaca isian JSON datar: kolom tak dikenal ditolak (sama dengan DecodeJSON biasa), lalu mengembalikan
// deret angka (nil = tidak diisi / null) dari kolom cols. known = nama isian tetap yang sah.
func SplitInput(b []byte, known []string, cols []string) ([]*float64, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	ok := make(map[string]bool, len(known)+len(cols))
	for _, k := range known {
		ok[k] = true
	}
	for _, c := range cols {
		ok[c] = true
	}
	for k := range raw {
		if !ok[k] {
			return nil, fmt.Errorf("json: unknown field %q", k)
		}
	}
	out := make([]*float64, len(cols))
	for i, c := range cols {
		v, has := raw[c]
		if !has {
			continue
		}
		var f *float64
		if err := json.Unmarshal(v, &f); err != nil {
			return nil, fmt.Errorf("%s harus berupa angka", c)
		}
		out[i] = f
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Validasi isian
// ---------------------------------------------------------------------

// Text: teks wajib (setelah dipangkas) dengan batas panjang.
func Text(p *string, col string, max int, errs *[]dailyinput.RowIssue) string {
	v, ok := dailyinput.Required(p, col, errs)
	if ok && utf8.RuneCountInString(v) > max {
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: fmt.Sprintf("%s maks. %d karakter", col, max)})
	}
	return v
}

// Year: tahun wajib, MinYear..MaxYear.
func Year(p *int, errs *[]dailyinput.RowIssue) int {
	switch {
	case p == nil:
		*errs = append(*errs, dailyinput.RowIssue{Column: "plan_year", Message: "plan_year wajib diisi"})
		return 0
	case *p < MinYear || *p > MaxYear:
		*errs = append(*errs, dailyinput.RowIssue{Column: "plan_year", Message: fmt.Sprintf("plan_year harus %d sampai %d", MinYear, MaxYear)})
		return 0
	}
	return *p
}

// Month: bulan wajib, 1..12.
func Month(p *int, errs *[]dailyinput.RowIssue) int {
	switch {
	case p == nil:
		*errs = append(*errs, dailyinput.RowIssue{Column: "plan_month", Message: "plan_month wajib diisi"})
		return 0
	case *p < 1 || *p > 12:
		*errs = append(*errs, dailyinput.RowIssue{Column: "plan_month", Message: "plan_month harus 1 sampai 12"})
		return 0
	}
	return *p
}

// Amount: satu angka ≥ 0 dibulatkan 2 desimal. Kosong → 0 (ok=false bila nil); NaN (sudah dilaporkan pengurai Excel) → 0.
func Amount(v *float64, col string, errs *[]dailyinput.RowIssue) (float64, bool) {
	switch {
	case v == nil || math.IsNaN(*v):
		return 0, false
	case *v < 0 || *v >= maxAmt:
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " harus 0 atau lebih"})
		return 0, false
	}
	return validate.Round2(*v), true
}

// Amounts: deret angka isian → angka valid (kosong → 0), urutan sama dengan cols.
func Amounts(in []*float64, cols []string, errs *[]dailyinput.RowIssue) []float64 {
	out := make([]float64, len(cols))
	for i, c := range cols {
		if i < len(in) {
			out[i], _ = Amount(in[i], c, errs)
		}
	}
	return out
}

// Sum = jumlah deret, dibulatkan 2 desimal.
func Sum(vals []float64) float64 {
	var s float64
	for _, v := range vals {
		s += v
	}
	return validate.Round2(s)
}

// Total: total dari isian bila ada (disimpan apa adanya), kosong → jumlah deret.
func Total(in *float64, sum float64, errs *[]dailyinput.RowIssue) float64 {
	if v, ok := Amount(in, "total", errs); ok {
		return v
	}
	return sum
}

// Whole: angka sel Excel → bilangan bulat (tahun/bulan). nil/NaN → nil; pecahan → kesalahan.
func Whole(v *float64, col string, errs *[]dailyinput.RowIssue) *int {
	if v == nil || math.IsNaN(*v) {
		return nil
	}
	if *v != math.Trunc(*v) {
		*errs = append(*errs, dailyinput.RowIssue{Column: col, Message: col + " harus bilangan bulat"})
		return nil
	}
	n := int(*v)
	return &n
}

// Pointers: salinan deret sebagai penunjuk (untuk menyusun isian dari data tersimpan).
func Pointers(vals []float64) []*float64 {
	out := make([]*float64, len(vals))
	for i := range vals {
		v := vals[i]
		out[i] = &v
	}
	return out
}

// MergeSeries menimpa base dengan setiap elemen patch yang tidak nil; changed=true bila ada yang ditimpa.
func MergeSeries(base, patch []*float64) (merged []*float64, changed bool) {
	merged = append([]*float64(nil), base...)
	for i := range patch {
		if i < len(merged) && patch[i] != nil {
			merged[i], changed = patch[i], true
		}
	}
	return merged, changed
}
