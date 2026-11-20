// Package http (fitur pagerule) = endpoint /api/v1/access (hak akses halaman & laporan).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/pagerule/domain"
	"simha/core-api/internal/pagerule/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/access/mine", h.Mine)
	api.GET("/access/rules", h.List)
	api.PUT("/access/rules", h.Save)
}

// GET /api/v1/access/mine — perubahan dari aturan bawaan untuk pemanggil di project aktifnya
func (h *Handler) Mine(c *gin.Context) {
	m, err := h.svc.Mine(c.Request.Context(), response.ActorFrom(c))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// GET /api/v1/access/rules?project_id= — seluruh aturan project (superadmin)
func (h *Handler) List(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// PUT /api/v1/access/rules — simpan perubahan satu role / satu user (superadmin)
func (h *Handler) Save(c *gin.Context) {
	var in domain.SaveInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	if response.WriteError(c, h.svc.Save(c.Request.Context(), response.ActorFrom(c), in)) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
