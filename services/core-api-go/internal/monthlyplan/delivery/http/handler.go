// Package http (fitur monthlyplan) = endpoint /api/v1/monthly-plans.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/monthlyplan/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (/imports/monthly-plans) ada di fitur importexcel.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/monthly-plans", h.List)
	api.POST("/monthly-plans", h.Set)
	api.PATCH("/monthly-plans/:id", h.Update)
	api.DELETE("/monthly-plans/:id", h.Delete)
}

// GET /api/v1/monthly-plans?project_id=&year=&month=&partner=
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

// POST /api/v1/monthly-plans — mitra + tahun + bulan + deskripsi + nama sama → diperbarui
func (h *Handler) Set(c *gin.Context) {
	var in service.MonthlyPlanInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	p, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": p, "created": created})
}

// PATCH /api/v1/monthly-plans/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.MonthlyPlanInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	p, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

// DELETE /api/v1/monthly-plans/:id
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
