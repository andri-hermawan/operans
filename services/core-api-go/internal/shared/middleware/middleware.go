// Package middleware = "penjaga" yang berjalan sebelum handler: ID per request, log akses,
// dan pemeriksaan access token (RequireAccess).
package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/core-api/internal/shared/token"
)

// Kunci nilai yang disimpan di gin.Context (dibaca lagi oleh shared/response & handler).
const (
	RequestIDKey = "request_id"
	ClaimsKey    = "claims" // *token.AccessClaims dari RequireAccess
)

// RequestID memberi ID unik tiap request, supaya satu request bisa dilacak di semua baris log.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// AccessLog mencatat satu baris per request. Header (token, password) TIDAK pernah dicatat.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http",
			"request_id", c.GetString(RequestIDKey),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
	}
}

// RequireAccess = middleware: wajib access token core-api yang sah.
func RequireAccess(signer *token.Signer) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token diperlukan (Authorization: Bearer <token>)"})
			return
		}
		claims, err := signer.Verify(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}
