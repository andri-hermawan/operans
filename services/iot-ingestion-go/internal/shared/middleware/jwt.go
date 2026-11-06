package middleware

import (
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CtxPrincipal = kunci context Gin tempat JWTAuth menitipkan Principal (dibaca lewat response.Principal).
const CtxPrincipal = "principal"

// Role per project (ADR 0002). Superadmin ditandai klaim "sa", bukan role project.
const (
	RoleAdmin      = "admin"
	RoleDispatcher = "dispatcher"
	RoleViewer     = "viewer"
)

var validRoles = map[string]bool{RoleAdmin: true, RoleDispatcher: true, RoleViewer: true}

// AccessClaims = isi access token buatan core-api.
type AccessClaims struct {
	Type        string `json:"typ"`   // harus "access"
	Name        string `json:"name"`  // nama user (untuk tampilan/log)
	ProjectID   string `json:"pid"`   // project aktif
	ProjectRole string `json:"prole"` // role di project aktif
	SuperAdmin  bool   `json:"sa"`    // akses lintas project
	jwt.RegisteredClaims
}

// Principal = SIAPA yang sedang memanggil API, diambil dari token yang sudah diverifikasi.
type Principal struct {
	UserID      uuid.UUID
	Name        string
	ProjectID   uuid.UUID
	ProjectRole string
	SuperAdmin  bool
}

// ProjectScope: hanya project aktif di token. nil (semua project) HANYA untuk superadmin yang belum memilih project.
func (p Principal) ProjectScope() *uuid.UUID {
	if p.SuperAdmin && p.ProjectID == uuid.Nil {
		return nil
	}
	return &p.ProjectID
}

// LoadPublicKey membaca kunci PUBLIK RS256 (format PEM) dari file.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM(pemBytes)
}

type JWTOptions struct {
	PublicKey *rsa.PublicKey
	Issuer    string // harus sama dengan "iss" di token (simha-core)
	Audience  string // harus ada di "aud" token (simha-api)
}

var errInvalidToken = errors.New("token tidak valid atau kedaluwarsa")

// JWTVerifier memeriksa access token. Dipakai middleware HTTP (header Authorization)
// dan WebSocket (token dikirim di pesan pertama, karena browser tidak bisa memasang header).
type JWTVerifier struct {
	parser  *jwt.Parser
	keyFunc jwt.Keyfunc
}

func NewJWTVerifier(opt JWTOptions) *JWTVerifier {
	return &JWTVerifier{
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256"}), // tolak HS256, "none", dll.
			jwt.WithExpirationRequired(),            // token tanpa exp ditolak
			jwt.WithIssuer(opt.Issuer),              // hanya token dari core-api
			jwt.WithAudience(opt.Audience),          // hanya token yang memang untuk API SIMHA
			jwt.WithLeeway(30*time.Second),          // toleransi selisih jam antar-server
		),
		keyFunc: func(*jwt.Token) (any, error) { return opt.PublicKey, nil },
	}
}

// Verify mengembalikan pemanggil (Principal) dan waktu kedaluwarsa token.
func (v *JWTVerifier) Verify(raw string) (Principal, time.Time, error) {
	var claims AccessClaims
	tok, err := v.parser.ParseWithClaims(raw, &claims, v.keyFunc)
	if err != nil || !tok.Valid {
		return Principal{}, time.Time{}, errInvalidToken
	}
	p, err := principalFrom(&claims)
	if err != nil {
		return Principal{}, time.Time{}, err
	}
	return p, claims.ExpiresAt.Time, nil
}

// JWTAuth memverifikasi access token buatan core-api, lalu menyimpan Principal di context.
func JWTAuth(opt JWTOptions) gin.HandlerFunc {
	v := NewJWTVerifier(opt)
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token diperlukan (Authorization: Bearer <token>)"})
			return
		}
		p, _, err := v.Verify(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set(CtxPrincipal, p)
		c.Next()
	}
}

// principalFrom memeriksa klaim khusus SIMHA (tanda tangan & waktu sudah dicek parser).
func principalFrom(cl *AccessClaims) (Principal, error) {
	if cl.Type != "access" {
		return Principal{}, errors.New("bukan access token")
	}
	userID, err := uuid.Parse(cl.Subject)
	if err != nil {
		return Principal{}, errors.New("klaim sub harus UUID user")
	}
	p := Principal{UserID: userID, Name: cl.Name, ProjectRole: cl.ProjectRole, SuperAdmin: cl.SuperAdmin}
	if p.SuperAdmin {
		if id, err := uuid.Parse(cl.ProjectID); err == nil {
			p.ProjectID = id // superadmin boleh punya project aktif, tapi tidak wajib
		}
		return p, nil
	}
	if p.ProjectID, err = uuid.Parse(cl.ProjectID); err != nil {
		return Principal{}, errors.New("klaim pid (project aktif) wajib berupa UUID")
	}
	if !validRoles[p.ProjectRole] {
		return Principal{}, errors.New("klaim prole tidak dikenal")
	}
	return p, nil
}
