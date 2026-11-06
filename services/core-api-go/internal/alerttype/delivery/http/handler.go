// Package http (fitur alerttype) = endpoint /api/v1/alert-types.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/alerttype/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/alert-types", h.List)
	api.PATCH("/alert-types/:code", h.Update)
}

// GET /api/v1/alert-types — jenis alert + nilai bawaan
func (h *Handler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// PATCH /api/v1/alert-types/:code — ubah nilai bawaan (superadmin)
func (h *Handler) Update(c *gin.Context) {
	var in service.AlertTypeInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	t, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), c.Param("code"), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}
