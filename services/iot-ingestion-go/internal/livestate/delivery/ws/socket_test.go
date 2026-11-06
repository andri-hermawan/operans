package ws

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/shared/middleware"
)

func TestLiveSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	hub := NewHub(10, 8)
	r := gin.New()
	r.GET("/ws/live", LiveSocket(LiveSocketConfig{
		Hub:        hub,
		Verifier:   middleware.NewJWTVerifier(middleware.JWTOptions{PublicKey: &key.PublicKey, Issuer: testIssuer, Audience: testAudience}),
		CheckEvery: 200 * time.Millisecond, // dipercepat untuk test (aslinya 30 detik)
	}))
	srv := httptest.NewServer(r)
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/live"

	projectA := uuid.MustParse(testProject)
	projectB := uuid.New()

	connect := func(t *testing.T, authMsg string) (*websocket.Conn, context.Context) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)
		conn, _, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		conn.Write(ctx, websocket.MessageText, []byte(authMsg))
		return conn, ctx
	}
	auth := func(token string) string {
		b, _ := json.Marshal(map[string]string{"type": "auth", "token": token})
		return string(b)
	}
	readType := func(t *testing.T, ctx context.Context, conn *websocket.Conn) (string, json.RawMessage) {
		t.Helper()
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("baca pesan: %v", err)
		}
		var env struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		json.Unmarshal(data, &env)
		return env.Type, env.Data
	}
	expectClose := func(t *testing.T, ctx context.Context, conn *websocket.Conn, want websocket.StatusCode) {
		t.Helper()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				if got := websocket.CloseStatus(err); got != want {
					t.Fatalf("kode penutupan = %d, seharusnya %d (err: %v)", got, want, err)
				}
				return
			}
		}
	}

	t.Run("pesan pertama bukan auth → ditutup 4401", func(t *testing.T) {
		conn, ctx := connect(t, `{"type":"halo"}`)
		expectClose(t, ctx, conn, closeUnauthorized)
	})

	t.Run("token dari kunci lain → ditutup 4401", func(t *testing.T) {
		other, _ := rsa.GenerateKey(rand.Reader, 2048)
		conn, ctx := connect(t, auth(signRS256(t, other, claimsValid())))
		expectClose(t, ctx, conn, closeUnauthorized)
	})

	t.Run("token sah: ready, lalu hanya menerima data project sendiri", func(t *testing.T) {
		conn, ctx := connect(t, auth(signRS256(t, key, claimsValid())))
		if typ, _ := readType(t, ctx, conn); typ != TypeReady {
			t.Fatalf("pesan pertama = %q, seharusnya ready", typ)
		}
		hub.Publish(&projectB, TypeEquipmentState, map[string]string{"unit": "B"}) // bukan project-nya
		hub.Publish(&projectA, TypeEquipmentState, map[string]string{"unit": "A"})
		typ, data := readType(t, ctx, conn)
		if typ != TypeEquipmentState || !strings.Contains(string(data), `"A"`) {
			t.Fatalf("seharusnya menerima data unit A saja, dapat %s %s", typ, data)
		}
	})

	t.Run("token kedaluwarsa tanpa perpanjangan → ditutup 4401", func(t *testing.T) {
		short := with(map[string]any{"exp": time.Now().Add(time.Second).Unix()})
		conn, ctx := connect(t, auth(signRS256(t, key, short)))
		readType(t, ctx, conn) // ready
		expectClose(t, ctx, conn, closeUnauthorized)
	})

	t.Run("server berhenti → ditutup 1001 (going away)", func(t *testing.T) {
		conn, ctx := connect(t, auth(signRS256(t, key, claimsValid())))
		readType(t, ctx, conn) // ready
		hub.Close()
		expectClose(t, ctx, conn, websocket.StatusGoingAway)
	})
}

// Pembantu token uji (sama dengan di shared/middleware/jwt_test.go).
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
