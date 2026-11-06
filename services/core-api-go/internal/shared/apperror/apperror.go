// Package apperror = kesalahan yang dipakai BANYAK fitur. Lapisan HTTP (shared/response) menerjemahkan
// setiap kesalahan di sini ke kode HTTP yang sama di semua endpoint (mis. ErrNotFound → 404).
package apperror

import (
	"errors"
	"fmt"
)

// Kesalahan data (dulu di domain/users.go).
var (
	ErrEmailExists = errors.New("email sudah terdaftar")
	ErrCodeExists  = errors.New("kode equipment sudah dipakai di project ini")
	ErrDuplicate   = errors.New("data sudah ada") // dibungkus dengan keterangan kuncinya (fmt.Errorf("%w: …"))
	ErrNotFound    = errors.New("data tidak ditemukan")
)

// Kesalahan hak akses (dulu di service/users.go; diterjemahkan ke kode HTTP).
var (
	ErrForbidden = errors.New("anda tidak berhak melakukan ini")
	ErrSelf      = errors.New("tidak boleh mengubah hak akses / status akun anda sendiri")
)

// ErrInvalidRefresh = sesi tidak valid / akun tidak aktif (fitur auth; juga dipetakan ke 401 oleh response).
var ErrInvalidRefresh = errors.New("sesi tidak valid atau sudah berakhir, silakan login ulang")

// ErrImportHasErrors = simpan ditolak karena masih ada baris yang salah (HTTP 422, hasil tetap dikirim).
// Dipakai impor Excel (importexcel) dan upload peta (maparea).
var ErrImportHasErrors = errors.New("masih ada baris yang salah; perbaiki lalu unggah ulang. Tidak ada data yang disimpan")

// ValidationError = isian tidak valid (HTTP 400).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Invalid membuat ValidationError dengan format seperti fmt.Sprintf.
func Invalid(format string, a ...any) error { return &ValidationError{Msg: fmt.Sprintf(format, a...)} }
