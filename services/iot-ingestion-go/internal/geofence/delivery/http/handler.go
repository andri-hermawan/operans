// Package http = pintu HTTP fitur geofence (halaman Geofencing).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/geofence/service"
	"simha/iot-ingestion/internal/shared/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur geofence (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/geofence-events", h.GeofenceEvents)
}

// GET /api/v1/geofence-events?date&shift&segment&equipment_id
func (h *Handler) GeofenceEvents(c *gin.Context) {
	s, ok := response.ReadSlot(c)
	if !ok {
		return
	}
	rows, err := h.svc.List(c.Request.Context(), s, response.OptStr(c, "segment"))
	if response.InternalErr(c, err, "kejadian area gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}
