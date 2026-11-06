// Package http (fitur equipment) = endpoint /api/v1/equipment: baca request → panggil service → tulis JSON.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/equipment/domain"
	"simha/core-api/internal/equipment/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Catatan: /equipment/:id/fuel-calibration milik fitur fuelcalibration — nama parameter harus sama (:id).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/equipment", h.List)
	api.POST("/equipment", h.Create)
	api.GET("/equipment/:id", h.Get)
	api.PATCH("/equipment/:id", h.Update)
}

// GET /api/v1/equipment?q=&project_id=&include_inactive=true
func (h *Handler) List(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), pid, c.Query("q"), c.Query("include_inactive") == "true")
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// GET /api/v1/equipment/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), response.ActorFrom(c), id)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}

// POST /api/v1/equipment
func (h *Handler) Create(c *gin.Context) {
	var in domain.EquipmentInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	e, err := h.svc.Create(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": e})
}

// PATCH /api/v1/equipment/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	var in domain.EquipmentInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	e, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}
