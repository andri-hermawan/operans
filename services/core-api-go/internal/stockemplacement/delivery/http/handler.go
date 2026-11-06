// Package http (fitur stockemplacement) = endpoint /api/v1/stock-emplacements.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/stockemplacement/service"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (/imports/stock-emplacements) ada di fitur importexcel.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/stock-emplacements", h.List)
	api.POST("/stock-emplacements", h.Set)
	api.PATCH("/stock-emplacements/:id", h.Update)
	api.DELETE("/stock-emplacements/:id", h.Delete)
}

// GET /api/v1/stock-emplacements?project_id=&date_from=&date_to=
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

// POST /api/v1/stock-emplacements — tanggal sama → diperbarui
func (h *Handler) Set(c *gin.Context) {
	var in service.StockEmplacementInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	e, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": e, "created": created})
}

// PATCH /api/v1/stock-emplacements/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.StockEmplacementInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	e, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}

// DELETE /api/v1/stock-emplacements/:id
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
