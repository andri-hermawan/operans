// Package http (fitur reference) = endpoint /api/v1/reference-values.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/reference/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/reference-values", h.List)
}

// GET /api/v1/reference-values?group=breakdown_category
func (h *Handler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("group"))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
