// Package http (fitur fuelfilling) = endpoint /api/v1/fuel-fillings.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/fuelfilling/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (/imports/fuel-fillings) ada di fitur importexcel.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/fuel-fillings", h.List)
	api.POST("/fuel-fillings", h.Set)
	api.PATCH("/fuel-fillings/:id", h.Update)
	api.DELETE("/fuel-fillings/:id", h.Delete)
}

// GET /api/v1/fuel-fillings?project_id=&date_from=&date_to=&shift_id=&equipment_code=
func (h *Handler) List(c *gin.Context) {
	f, ok := response.DailyFilter(c)
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), f)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/fuel-fillings — tanggal + shift + unit + HM pengisian sama → diperbarui
func (h *Handler) Set(c *gin.Context) {
	var in service.FuelFillingInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	f, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": f, "created": created})
}

// PATCH /api/v1/fuel-fillings/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.FuelFillingInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	f, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": f})
}

// DELETE /api/v1/fuel-fillings/:id
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
