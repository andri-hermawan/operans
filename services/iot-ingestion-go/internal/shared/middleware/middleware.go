// Package middleware berisi "penjaga" yang dijalankan Gin SEBELUM handler fitur:
//   - JWTAuth / JWTVerifier: memeriksa access token pengguna (RS256 buatan core-api) → Principal
//   - RequestID: memberi ID unik tiap request (untuk melacak log)
//   - AccessLog: satu baris log per request
//
// Pemeriksa API key alat (X-API-Key) hanya dipakai satu endpoint, jadi letaknya di fitur ingest
// (internal/ingest/delivery/http/auth.go).
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CtxRequestID = kunci context Gin tempat RequestID menitipkan ID request.
const CtxRequestID = "request_id"

// RequestID memberi ID unik tiap request, supaya satu request bisa dilacak di semua baris log.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Set(CtxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog mencatat satu baris per request. Header (API key, token) TIDAK pernah dicatat.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http",
			"request_id", c.GetString(CtxRequestID),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
	}
}
