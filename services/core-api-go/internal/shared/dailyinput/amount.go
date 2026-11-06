package dailyinput

import (
	"math"
)

// Amount memeriksa angka isian: kosong → nil; 0 atau lebih dan di bawah 1e10; dibulatkan ke `decimals` desimal.
// NaN = sudah dilaporkan pengurai Excel (tidak dilaporkan dua kali).
func Amount(v *float64, col string, decimals int, errs *[]RowIssue) *float64 {
	if v == nil || math.IsNaN(*v) {
		return nil
	}
	if *v < 0 || *v >= 1e10 {
		*errs = append(*errs, RowIssue{Column: col, Message: col + " harus 0 atau lebih"})
		return nil
	}
	p := math.Pow10(decimals)
	r := math.Round(*v*p) / p
	return &r
}
