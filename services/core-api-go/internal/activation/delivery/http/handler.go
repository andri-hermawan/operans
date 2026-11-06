// Package http (fitur activation) = endpoint /api/v1/equipment-activations.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/activation/domain"
	"simha/core-api/internal/activation/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/equipment-activations", h.List)
	api.POST("/equipment-activations", h.Set)
	api.PATCH("/equipment-activations/:id", h.Update)
	api.DELETE("/equipment-activations/:id", h.Delete)
}

// GET /api/v1/equipment-activations?project_id=&equipment_id=&date_from=&date_to=&status=
func (h *Handler) List(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	eid, ok := response.QueryUUID(c, "equipment_id")
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), domain.ActivationFilter{
		ProjectID: pid, EquipmentID: eid,
		DateFrom: c.Query("date_from"), DateTo: c.Query("date_to"), Status: c.Query("status"),
	})
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/equipment-activations — simpan status equipment pada satu tanggal (ditimpa bila sudah ada)
func (h *Handler) Set(c *gin.Context) {
	var in service.ActivationInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	act, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusCreated
	}
	c.JSON(code, gin.H{"data": act, "created": created})
}

// PATCH /api/v1/equipment-activations/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathInt(c, "id")
	if !ok {
		return
	}
	var in service.ActivationInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	act, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": act})
}

// DELETE /api/v1/equipment-activations/:id
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
