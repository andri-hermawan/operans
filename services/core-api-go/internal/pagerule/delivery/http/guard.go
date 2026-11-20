package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/pagerule/domain"
	"simha/core-api/internal/shared/response"
)

// Guard = middleware untuk grup /api/v1: endpoint data Report dan Record / Upload hanya melayani user yang halamannya
// terbuka menurut hak akses (aturan bawaan + pengaturan superadmin). Endpoint lain lewat tanpa pemeriksaan.
func (h *Handler) Guard() gin.HandlerFunc {
	return func(c *gin.Context) {
		keys := domain.KeysFor(c.Request.URL.Path)
		if keys == nil {
			c.Next()
			return
		}
		ok, err := h.svc.Allowed(c.Request.Context(), response.ActorFrom(c), keys)
		if err != nil {
			response.WriteError(c, err)
			c.Abort()
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "anda tidak punya akses ke halaman atau laporan ini"})
			return
		}
		c.Next()
	}
}
