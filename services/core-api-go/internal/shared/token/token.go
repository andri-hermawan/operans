// Package token = membuat & memeriksa access token JWT RS256 (+ JWKS) dan refresh token acak.
// Dipakai fitur auth (login), middleware (cek token), dan reportschedule (token sementara halaman cetak).
package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessClaims = isi access token (ADR 0002). Layanan lain (iot, fastapi) memeriksa isi yang sama.
type AccessClaims struct {
	Type        string `json:"typ"`             // selalu "access"
	Name        string `json:"name"`            // nama user
	ProjectID   string `json:"pid,omitempty"`   // project aktif (kosong: superadmin tanpa project)
	ProjectRole string `json:"prole,omitempty"` // admin | dispatcher | viewer
	SuperAdmin  bool   `json:"sa"`
	jwt.RegisteredClaims
}

// Signer membuat & memeriksa access token dengan kunci PRIVAT RS256 milik core-api.
type Signer struct {
	key       *rsa.PrivateKey
	kid       string // "key id": membantu layanan lain memilih kunci bila kelak kunci diganti
	issuer    string
	audience  string
	accessTTL time.Duration
	now       func() time.Time
	parser    *jwt.Parser
}

func NewSigner(key *rsa.PrivateKey, issuer, audience string, accessTTL time.Duration) (*Signer, error) {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(der)
	return &Signer{
		key: key, kid: hex.EncodeToString(sum[:8]),
		issuer: issuer, audience: audience, accessTTL: accessTTL, now: time.Now,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithExpirationRequired(),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
			jwt.WithLeeway(30*time.Second),
		),
	}, nil
}

// LoadPrivateKey membaca kunci PRIVAT RS256 (PEM) dari file. File ini RAHASIA.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
}

// AccessSubject = data yang dimasukkan ke access token.
type AccessSubject struct {
	UserID      uuid.UUID
	Name        string
	ProjectID   *uuid.UUID
	ProjectRole string
	SuperAdmin  bool
}

// IssueAccess membuat access token baru (umur pendek, bawaan 15 menit).
func (s *Signer) IssueAccess(sub AccessSubject) (token string, expiresAt time.Time, err error) {
	now := s.now()
	expiresAt = now.Add(s.accessTTL)
	claims := AccessClaims{
		Type: "access", Name: sub.Name, ProjectRole: sub.ProjectRole, SuperAdmin: sub.SuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings{s.audience},
			Subject:   sub.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	}
	if sub.ProjectID != nil {
		claims.ProjectID = sub.ProjectID.String()
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t.Header["kid"] = s.kid
	token, err = t.SignedString(s.key)
	return token, expiresAt, err
}

var ErrInvalidAccessToken = errors.New("token tidak valid atau kedaluwarsa")

// Verify memeriksa access token buatan core-api sendiri (dipakai endpoint seperti /auth/me).
func (s *Signer) Verify(raw string) (*AccessClaims, error) {
	var c AccessClaims
	tok, err := s.parser.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return &s.key.PublicKey, nil })
	if err != nil || !tok.Valid || c.Type != "access" {
		return nil, ErrInvalidAccessToken
	}
	return &c, nil
}

// JWKS = kunci PUBLIK dalam format standar JSON Web Key Set (RFC 7517),
// diterbitkan di /.well-known/jwks.json agar layanan lain bisa memeriksa token.
func (s *Signer) JWKS() map[string]any {
	pub := s.key.PublicKey
	return map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": s.kid,
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}}
}

// NewRefreshToken membuat refresh token acak (256 bit). raw dikirim ke klien SEKALI;
// database hanya menyimpan hash-nya.
func NewRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken = SHA-256 hex dari refresh token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
