// Package http (fitur shift) = endpoint /api/v1/shifts (pilihan shift & kelola shift).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/shift/domain"
	"simha/core-api/internal/shift/service"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/shifts/manage", h.ListAll)
	api.POST("/shifts", h.Save)
	api.PATCH("/shifts/:id", h.Save)
	api.GET("/shifts", h.Options)
}

// GET /api/v1/shifts?project_id=
func (h *Handler) Options(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.Options(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GET /api/v1/shifts/manage?project_id= — termasuk shift nonaktif
func (h *Handler) ListAll(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.ListAll(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/shifts · PATCH /api/v1/shifts/:id (admin project & superadmin)
func (h *Handler) Save(c *gin.Context) {
	id, ok := response.OptionalPathUUID(c)
	if !ok {
		return
	}
	var in domain.ShiftInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	s, err := h.svc.Save(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.SavedStatus(id), gin.H{"data": s})
}
