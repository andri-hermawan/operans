// Package http (fitur fuelcalibration) = endpoint kalibrasi fuel: /api/v1/fuel-calibrations dan
// /api/v1/equipment/:id/fuel-calibration (+ /lookup).
package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/fuelcalibration/service"
	"simha/core-api/internal/shared/response"
)

// Handler = kalibrasi fuel (B7, ADR 0008).
type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Nama parameter :id harus sama dengan route /equipment/:id milik fitur equipment.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/fuel-calibrations", h.List)
	api.GET("/equipment/:id/fuel-calibration", h.Get)
	api.PUT("/equipment/:id/fuel-calibration", h.Set)
	api.DELETE("/equipment/:id/fuel-calibration", h.Delete)
	api.GET("/equipment/:id/fuel-calibration/lookup", h.Lookup)
}

// GET /api/v1/fuel-calibrations?project_id= — ringkasan kalibrasi semua equipment project
func (h *Handler) List(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GET /api/v1/equipment/:id/fuel-calibration — kalibrasi + semua titik
func (h *Handler) Get(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	cal, err := h.svc.Get(c.Request.Context(), response.ActorFrom(c), id)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cal})
}

// PUT /api/v1/equipment/:id/fuel-calibration — ganti seluruh kalibrasi (admin)
func (h *Handler) Set(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	var in service.FuelCalibrationInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	cal, created, err := h.svc.Set(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.CreatedStatus(created), gin.H{"data": cal, "created": created})
}

// DELETE /api/v1/equipment/:id/fuel-calibration
func (h *Handler) Delete(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	if response.WriteError(c, h.svc.Delete(c.Request.Context(), response.ActorFrom(c), id)) {
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /api/v1/equipment/:id/fuel-calibration/lookup?value= — nilai sensor → liter & persen
func (h *Handler) Lookup(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	v, err := strconv.ParseFloat(c.Query("value"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value wajib berupa angka (nilai sensor)"})
		return
	}
	r, err := h.svc.Lookup(c.Request.Context(), response.ActorFrom(c), id, v)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": r})
}
