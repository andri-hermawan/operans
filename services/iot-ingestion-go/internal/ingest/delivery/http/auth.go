package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/shared/response"
)

const ctxDevice = "device"

// DeviceFinder = pencari alat dari API key (dipenuhi *service.Service milik fitur device).
type DeviceFinder interface {
	FindDeviceByAPIKey(ctx context.Context, key string) (*devicedomain.Device, error)
}

// DeviceAuth: baca header X-API-Key → cari alat → simpan alat di context request.
func DeviceAuth(devices DeviceFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-API-Key diperlukan"})
			return
		}
		dev, err := devices.FindDeviceByAPIKey(c.Request.Context(), key)
		if err != nil {
			slog.Error("cek API key gagal", "request_id", response.RequestID(c), "err", err) // JANGAN log key-nya
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		if dev == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key tidak dikenal atau alat nonaktif"})
			return
		}
		c.Set(ctxDevice, dev)
		c.Next()
	}
}
