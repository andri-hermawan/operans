// Package http = pintu HTTP fitur telemetry: latest/history satu equipment, titik riwayat & heatmap,
// speed per segmen, ringkasan aktivitas. Daftar route ada di Register.
package http

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/shared/response"
	"simha/iot-ingestion/internal/telemetry/service"
)

const maxPoints = 50000

type Handler struct {
	svc      *service.Service
	maxRange time.Duration // batas rentang history (MAX_HISTORY_RANGE_HOURS)
	maxRows  int           // batas jumlah baris history (MAX_HISTORY_ROWS)
}

func New(svc *service.Service, maxRange time.Duration, maxRows int) *Handler {
	return &Handler{svc: svc, maxRange: maxRange, maxRows: maxRows}
}

// Register mendaftarkan route fitur telemetry (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/equipment/:equipmentId/latest", h.Latest)
	api.GET("/equipment/:equipmentId/history", h.History)

	// Pemantauan (F3/F4): data per tanggal operasional + shift
	api.GET("/telemetry/points", h.Points)
	api.GET("/telemetry/segment-speeds", h.SegmentSpeeds)
	api.GET("/telemetry/heat-grid", h.HeatGrid)
	api.GET("/activity-summary", h.ActivitySummary)
}

// Latest: data terbaru satu equipment, hanya dari project yang boleh dilihat pemanggil.
func (h *Handler) Latest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("equipmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "equipmentId bukan UUID"})
		return
	}
	row, err := h.svc.Latest(c.Request.Context(), response.Principal(c).ProjectScope(), id)
	if err != nil {
		slog.Error("latest gagal", "request_id", response.RequestID(c), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if row == nil {
		// Sama untuk "belum ada data" dan "milik project lain": tidak membocorkan keberadaan data.
		c.JSON(http.StatusNotFound, gin.H{"error": "belum ada data untuk equipment ini"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": row, "data_as_of": row.EventTime})
}

// History: riwayat satu equipment. Wajib from & to (RFC3339), rentang dan jumlah baris dibatasi.
func (h *Handler) History(c *gin.Context) {
	id, err := uuid.Parse(c.Param("equipmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "equipmentId bukan UUID"})
		return
	}
	from, err1 := time.Parse(time.RFC3339, c.Query("from"))
	to, err2 := time.Parse(time.RFC3339, c.Query("to"))
	if err1 != nil || err2 != nil || !to.After(from) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from & to wajib format RFC3339 (mis. 2026-10-22T00:00:00+07:00) dan to > from"})
		return
	}
	if to.Sub(from) > h.maxRange {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rentang waktu maksimal " + h.maxRange.String()})
		return
	}
	limit := h.maxRows
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v < limit {
		limit = v
	}

	scope := response.Principal(c).ProjectScope()
	rows, err := h.svc.History(c.Request.Context(), scope, id, from.UTC(), to.UTC(), limit)
	if err != nil {
		slog.Error("history gagal", "request_id", response.RequestID(c), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      rows,
		"count":     len(rows),
		"truncated": len(rows) == limit, // true = mungkin masih ada data lain; persempit rentang
		"scope": gin.H{
			"equipment_id": id, "project_id": scope, // project_id null = semua project (superadmin)
			"from": from.UTC(), "to": to.UTC(), "limit": limit,
		},
	})
}

// GET /api/v1/telemetry/points?date&shift&equipment_id&step_seconds=&limit=
// Tanpa equipment_id wajib step_seconds ≥ 30 (seluruh unit, heatmap) agar data tetap ringan.
func (h *Handler) Points(c *gin.Context) {
	s, ok := response.ReadSlot(c)
	if !ok {
		return
	}
	step, _ := strconv.Atoi(c.Query("step_seconds"))
	if s.EquipmentID == nil && step < 30 {
		step = 60
	}
	limit := maxPoints
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v < limit {
		limit = v
	}
	rows, err := h.svc.Points(c.Request.Context(), s, step, limit)
	if response.InternalErr(c, err, "titik telemetry gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows), "truncated": len(rows) == limit, "step_seconds": step})
}

// GET /api/v1/telemetry/segment-speeds?date&shift
func (h *Handler) SegmentSpeeds(c *gin.Context) {
	s, ok := response.ReadSlot(c)
	if !ok {
		return
	}
	rows, err := h.svc.SegmentSpeeds(c.Request.Context(), s)
	if response.InternalErr(c, err, "speed per segmen gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}

// GET /api/v1/activity-summary?date&shift&equipment_id — satu baris per unit
func (h *Handler) ActivitySummary(c *gin.Context) {
	s, ok := response.ReadSlot(c)
	if !ok {
		return
	}
	rows, err := h.svc.Activities(c.Request.Context(), s)
	if response.InternalErr(c, err, "ringkasan aktivitas gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}
