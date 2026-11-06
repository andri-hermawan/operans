// Package http (fitur yearlyplan) = endpoint /api/v1/yearly-plans.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/yearlyplan/service"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (/imports/yearly-plans) ada di fitur importexcel.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/yearly-plans", h.List)
	api.POST("/yearly-plans", h.Set)
	api.PATCH("/yearly-plans/:id", h.Update)
	api.DELETE("/yearly-plans/:id", h.Delete)
}

// GET /api/v1/yearly-plans?project_id=&year=&partner=
func (h *Handler) List(c *gin.Context) {
	f, ok := response.PlanFilter(c)
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), f)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/yearly-plans — mitra + tahun + lokasi + deskripsi + nama sama → diperbarui
func (h *Handler) Set(c *gin.Context) {
	var in service.YearlyPlanInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	p, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": p, "created": created})
}

// PATCH /api/v1/yearly-plans/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.YearlyPlanInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	p, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

// DELETE /api/v1/yearly-plans/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	if response.WriteError(c, h.svc.Delete(c.Request.Context(), response.ActorFrom(c), id)) {
		return
	}
	c.Status(http.StatusNoContent)
}
