// Package hash = hash password: argon2id untuk password baru + bcrypt (hash dari sistem lama).
// Tidak tahu soal database, HTTP, maupun SIMHA — bisa dipakai di program mana pun.
package hash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Parameter argon2id (rekomendasi OWASP): memori 64 MB, 3 iterasi, 2 thread.
// Bila kelak dinaikkan, hash lama otomatis diganti saat user login (lihat needsRehash).
const (
	argonMemory  = 64 * 1024
	argonTime    = 3
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

const (
	MinPasswordLen = 8
	MaxPasswordLen = 128
)

var ErrWeakPassword = fmt.Errorf("password minimal %d dan maksimal %d karakter", MinPasswordLen, MaxPasswordLen)

// ValidatePassword memeriksa aturan password baru (dipakai saat membuat/mengganti password).
func ValidatePassword(pw string) error {
	if n := utf8.RuneCountInString(pw); n < MinPasswordLen || n > MaxPasswordLen {
		return ErrWeakPassword
	}
	return nil
}

// HashPassword menghasilkan string argon2id format standar (PHC):
// $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
func HashPassword(pw string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64(salt), b64(key)), nil
}

// VerifyPassword membandingkan password dengan hash tersimpan.
//   - ok          = password cocok
//   - needsRehash = hash tersimpan adalah bcrypt (sistem lama) atau argon2id berparameter lama,
//     jadi sebaiknya disimpan ulang dengan HashPassword setelah login berhasil.
func VerifyPassword(encoded, pw string) (ok, needsRehash bool) {
	switch {
	case strings.HasPrefix(encoded, "$2a$"), strings.HasPrefix(encoded, "$2b$"), strings.HasPrefix(encoded, "$2y$"):
		return bcrypt.CompareHashAndPassword([]byte(encoded), []byte(pw)) == nil, true
	case strings.HasPrefix(encoded, "$argon2id$"):
		m, t, p, salt, want, err := parseArgon2(encoded)
		if err != nil {
			return false, false
		}
		got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
		ok = subtle.ConstantTimeCompare(got, want) == 1 // waktu perbandingan tidak membocorkan isi hash
		return ok, ok && (m != argonMemory || t != argonTime || p != argonThreads)
	}
	return false, false
}

// DummyVerify menjalankan perhitungan yang sama beratnya walaupun user tidak ditemukan,
// agar penyerang tidak bisa menebak email mana yang terdaftar dari lamanya jawaban.
func DummyVerify(pw string) {
	dummyOnce.Do(func() { dummyHash, _ = HashPassword("simha-dummy-password") })
	VerifyPassword(dummyHash, pw)
}

var (
	dummyOnce sync.Once
	dummyHash string
)

func parseArgon2(encoded string) (m, t uint32, p uint8, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$") // ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]
	if len(parts) != 6 {
		return 0, 0, 0, nil, nil, errors.New("format argon2id tidak dikenal")
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	return
}

func b64(b []byte) string { return base64.RawStdEncoding.EncodeToString(b) }
