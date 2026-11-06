// Package http (fitur workhour) = endpoint /api/v1/work-hour-notifications.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/workhour/service"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (/imports/work-hour-notifications) ada di fitur importexcel.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/work-hour-notifications", h.List)
	api.POST("/work-hour-notifications", h.Set)
	api.PATCH("/work-hour-notifications/:id", h.Update)
	api.DELETE("/work-hour-notifications/:id", h.Delete)
}

// GET /api/v1/work-hour-notifications?project_id=&date_from=&date_to=&shift_id=&equipment_code=
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

// POST /api/v1/work-hour-notifications — unit + kategori + jam mulai sama → diperbarui
func (h *Handler) Set(c *gin.Context) {
	var in service.WorkHourInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	t, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": t, "created": created})
}

// PATCH /api/v1/work-hour-notifications/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.WorkHourInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	t, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}

// DELETE /api/v1/work-hour-notifications/:id
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
