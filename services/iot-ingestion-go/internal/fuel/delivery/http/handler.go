// Package http = pintu HTTP fitur fuel (penanda isi/turun BBM di Fuel History).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/fuel/service"
	"simha/iot-ingestion/internal/shared/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur fuel (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/fuel-events", h.FuelEvents)
}

// GET /api/v1/fuel-events?date (atau date_from & date_to)&shift&equipment_id
func (h *Handler) FuelEvents(c *gin.Context) {
	f, ok := response.ReadRange(c)
	if !ok {
		return
	}
	rows, err := h.svc.List(c.Request.Context(), f)
	if response.InternalErr(c, err, "kejadian fuel gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}
