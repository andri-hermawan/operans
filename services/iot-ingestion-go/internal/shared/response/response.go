// Package response berisi pembantu yang dipakai handler HTTP SEMUA fitur:
//   - Principal: siapa pemanggil (dititipkan middleware.JWTAuth)
//   - InternalErr: catat galat server di log lalu jawab 500 {"error":"internal"}
//   - ReadSlot / ReadRange / OptStr / ValidDate: membaca filter standar dari query URL
package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/shared/middleware"
)

// Principal mengambil Principal yang dititipkan JWTAuth.
func Principal(c *gin.Context) middleware.Principal {
	return c.MustGet(middleware.CtxPrincipal).(middleware.Principal)
}

// RequestID = ID request dari middleware.RequestID (untuk log).
func RequestID(c *gin.Context) string {
	return c.GetString(middleware.CtxRequestID)
}

// InternalErr: bila err tidak nil → catat di log (dengan pesan msg) dan jawab 500. Kembalian true = sudah dijawab.
func InternalErr(c *gin.Context, err error, msg string) bool {
	if err == nil {
		return false
	}
	slog.Error(msg, "request_id", RequestID(c), "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	return true
}
