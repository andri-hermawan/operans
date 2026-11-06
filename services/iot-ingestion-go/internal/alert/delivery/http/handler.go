// Package http = pintu HTTP fitur alert (halaman Abnormal Alerts & Distribution Map).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/alert/service"
	"simha/iot-ingestion/internal/shared/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur alert (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/alerts", h.Alerts)
}

// GET /api/v1/alerts?date_from&date_to (atau date)&shift&type=a,b&equipment_id&limit
func (h *Handler) Alerts(c *gin.Context) {
	f, ok := response.ReadRange(c)
	if !ok {
		return
	}
	rows, err := h.svc.List(c.Request.Context(), f)
	if response.InternalErr(c, err, "daftar alert gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows), "truncated": len(rows) == f.Limit})
}
