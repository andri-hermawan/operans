package token

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s, err := NewSigner(key, "simha-core", "simha-api", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	user, project := uuid.New(), uuid.New()
	tok, exp, err := s.IssueAccess(AccessSubject{UserID: user, Name: "Dispatcher", ProjectID: &project, ProjectRole: "dispatcher"})
	if err != nil || time.Until(exp) > 15*time.Minute {
		t.Fatalf("issue: exp=%v err=%v", exp, err)
	}
	c, err := s.Verify(tok)
	if err != nil || c.Subject != user.String() || c.ProjectID != project.String() || c.ProjectRole != "dispatcher" || c.Type != "access" {
		t.Fatalf("isi token salah: %+v err=%v", c, err)
	}

	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	s2, _ := NewSigner(other, "simha-core", "simha-api", time.Minute)
	if _, err := s2.Verify(tok); err == nil {
		t.Fatal("token dari kunci lain harus ditolak")
	}

	jwks := s.JWKS()["keys"].([]map[string]string)
	if jwks[0]["kty"] != "RSA" || jwks[0]["e"] != "AQAB" || jwks[0]["kid"] == "" {
		t.Fatalf("JWKS tidak sesuai standar: %v", jwks[0])
	}
}

func TestRefreshToken(t *testing.T) {
	raw1, hash1, _ := NewRefreshToken()
	raw2, _, _ := NewRefreshToken()
	if raw1 == raw2 || len(raw1) < 40 || HashToken(raw1) != hash1 || len(hash1) != 64 {
		t.Fatal("refresh token harus acak, cukup panjang, dan hash-nya SHA-256 hex")
	}
}
