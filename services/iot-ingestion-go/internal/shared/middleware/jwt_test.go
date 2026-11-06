package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "simha-core"
	testAudience = "simha-api"
	testUser     = "0b10705e-0000-4000-8000-000000000001"
	testProject  = "9811d93d-0000-4000-8000-000000000002"
)

// claimsValid = isi token yang benar; tiap skenario mengubah satu hal.
func claimsValid() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": testIssuer, "aud": testAudience, "typ": "access",
		"sub": testUser, "name": "Tester", "pid": testProject, "prole": "viewer", "sa": false,
		"iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}
}

func with(changes map[string]any) jwt.MapClaims {
	c := claimsValid()
	for k, v := range changes {
		if v == nil {
			delete(c, k)
		} else {
			c[k] = v
		}
	}
	return c
}

func signRS256(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Kunci uji dibuat di memori: kunci "core-api" dan kunci "penyerang"
	coreKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	attackerKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	r := gin.New()
	r.GET("/rahasia",
		JWTAuth(JWTOptions{PublicKey: &coreKey.PublicKey, Issuer: testIssuer, Audience: testAudience}),
		func(c *gin.Context) {
			p := c.MustGet(CtxPrincipal).(Principal)
			c.JSON(http.StatusOK, gin.H{"project": p.ProjectID, "role": p.ProjectRole})
		})

	// Serangan "key confusion": token HS256 yang ditandatangani memakai kunci PUBLIK sebagai secret
	pubDER, _ := x509.MarshalPKIXPublicKey(&coreKey.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	confused, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsValid()).SignedString(pubPEM)

	unsignedNone, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claimsValid()).
		SignedString(jwt.UnsafeAllowNoneSignatureType)

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"token valid (viewer)", "Bearer " + signRS256(t, coreKey, claimsValid()), http.StatusOK},
		{"superadmin tanpa pid", "Bearer " + signRS256(t, coreKey, with(map[string]any{"sa": true, "pid": nil, "prole": nil})), http.StatusOK},
		{"tanpa header", "", http.StatusUnauthorized},
		{"ditandatangani kunci lain", "Bearer " + signRS256(t, attackerKey, claimsValid()), http.StatusUnauthorized},
		{"sudah kedaluwarsa", "Bearer " + signRS256(t, coreKey, with(map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})), http.StatusUnauthorized},
		{"tanpa exp", "Bearer " + signRS256(t, coreKey, with(map[string]any{"exp": nil})), http.StatusUnauthorized},
		{"issuer salah", "Bearer " + signRS256(t, coreKey, with(map[string]any{"iss": "aplikasi-lain"})), http.StatusUnauthorized},
		{"audience salah", "Bearer " + signRS256(t, coreKey, with(map[string]any{"aud": "api-lain"})), http.StatusUnauthorized},
		{"bukan access token", "Bearer " + signRS256(t, coreKey, with(map[string]any{"typ": "refresh"})), http.StatusUnauthorized},
		{"tanpa pid", "Bearer " + signRS256(t, coreKey, with(map[string]any{"pid": nil})), http.StatusUnauthorized},
		{"role tidak dikenal", "Bearer " + signRS256(t, coreKey, with(map[string]any{"prole": "hacker"})), http.StatusUnauthorized},
		{"serangan alg=none", "Bearer " + unsignedNone, http.StatusUnauthorized},
		{"serangan key confusion (HS256)", "Bearer " + confused, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/rahasia", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Fatalf("status = %d, seharusnya %d (body: %s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
