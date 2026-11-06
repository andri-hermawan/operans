// Package http (fitur company) = endpoint /api/v1/companies: baca request → panggil service → tulis JSON.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/company/domain"
	"simha/core-api/internal/company/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/companies", h.List)
	api.POST("/companies", h.Save)
	api.PATCH("/companies/:id", h.Save)
}

// GET /api/v1/companies — superadmin: semua; admin: company project aktifnya
func (h *Handler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/companies · PATCH /api/v1/companies/:id (superadmin)
func (h *Handler) Save(c *gin.Context) {
	id, ok := response.OptionalPathUUID(c)
	if !ok {
		return
	}
	var in domain.CompanyInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	if response.WriteError(c, h.svc.Save(c.Request.Context(), response.ActorFrom(c), id, in)) {
		return
	}
	c.Status(response.SavedStatus(id))
}
