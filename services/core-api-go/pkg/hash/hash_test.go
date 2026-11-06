package hash

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	h, err := HashPassword("rahasia-panjang")
	if err != nil || !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash = %q, err = %v", h, err)
	}
	if ok, rehash := VerifyPassword(h, "rahasia-panjang"); !ok || rehash {
		t.Fatalf("password benar: ok=%v rehash=%v", ok, rehash)
	}
	if ok, _ := VerifyPassword(h, "salah"); ok {
		t.Fatal("password salah seharusnya ditolak")
	}
	h2, _ := HashPassword("rahasia-panjang")
	if h == h2 {
		t.Fatal("salt acak: dua hash dari password sama harus berbeda")
	}
}

func TestPasswordBcryptLama(t *testing.T) {
	old, _ := bcrypt.GenerateFromPassword([]byte("password-lama"), bcrypt.MinCost) // seperti hash $2b$ di fms_lite
	if ok, rehash := VerifyPassword(string(old), "password-lama"); !ok || !rehash {
		t.Fatalf("bcrypt lama harus diterima dan ditandai untuk diganti: ok=%v rehash=%v", ok, rehash)
	}
	if ok, _ := VerifyPassword(string(old), "salah"); ok {
		t.Fatal("password salah seharusnya ditolak")
	}
}

func TestValidatePassword(t *testing.T) {
	if ValidatePassword("pendek") == nil || ValidatePassword("cukup-panjang") != nil {
		t.Fatal("aturan panjang password tidak berjalan")
	}
}
