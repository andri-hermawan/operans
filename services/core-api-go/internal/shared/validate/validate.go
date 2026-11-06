// Package validate = fungsi kecil untuk memeriksa & merapikan isian (tanggal, kode, nama, angka)
// yang dipakai beberapa fitur. Kesalahan dikembalikan sebagai apperror.ValidationError (HTTP 400).
package validate

import (
	"math"
	"regexp"
	"strings"
	"time"

	"simha/core-api/internal/shared/apperror"
)

// IsDate: format YYYY-MM-DD.
func IsDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// TrimPtr memangkas spasi teks; nil tetap nil.
func TrimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	return &v
}

// Round2 membulatkan ke 2 desimal.
func Round2(f float64) float64 { return math.Round(f*100) / 100 }

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,49}$`)

// Code: kode huruf besar (company_code, project_code, shift_code). nil = tidak diisi.
func Code(p *string, field string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	v := strings.ToUpper(strings.TrimSpace(*p))
	if !codePattern.MatchString(v) {
		return nil, apperror.Invalid("%s hanya huruf besar, angka, _ atau - (maks. 50)", field)
	}
	return &v, nil
}

// Name: nama wajib (tidak kosong) dengan batas panjang. nil = tidak diisi.
func Name(p *string, field string, max int) (*string, error) {
	if p == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*p)
	if v == "" || len([]rune(v)) > max {
		return nil, apperror.Invalid("%s wajib diisi (maks. %d karakter)", field, max)
	}
	return &v, nil
}
