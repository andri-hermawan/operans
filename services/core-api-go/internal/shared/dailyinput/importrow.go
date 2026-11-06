package dailyinput

import (
	"math"
	"strings"

	"simha/core-api/internal/shared/excel"
)

// ---------------------------------------------------------------------
// Bantuan pengurai baris Excel (dipakai service/import.go tiap fitur)
// ---------------------------------------------------------------------

func PtrStr(s string) *string { return &s }

// Records mengubah []any (hasil pengurai baris) kembali ke jenis aslinya.
func Records[T any](recs []any) []T {
	out := make([]T, len(recs))
	for i, r := range recs {
		out[i] = r.(T)
	}
	return out
}

// RenameColumns mengganti nama isian API (mis. equipment_code) dengan judul kolom template (Asset ID).
func RenameColumns(errs []RowIssue, names map[string]string) []RowIssue {
	pairs := make([]string, 0, len(names)*2)
	for api, col := range names {
		pairs = append(pairs, api, col)
	}
	r := strings.NewReplacer(pairs...)
	for i := range errs {
		if col, ok := names[errs[i].Column]; ok {
			errs[i].Column = col
		}
		errs[i].Message = r.Replace(errs[i].Message)
	}
	return errs
}

// NumCell: angka dari sel (kosong → nil; salah → kesalahan + NaN agar tidak dilaporkan "wajib diisi" lagi).
func NumCell(row []string, i int, col string, parse func(string) (*float64, error), errs *[]RowIssue) *float64 {
	v, err := parse(excel.Cell(row, i))
	if err != nil {
		*errs = append(*errs, RowIssue{Column: col, Message: col + ": " + err.Error()})
		nan := math.NaN()
		return &nan
	}
	return v
}
