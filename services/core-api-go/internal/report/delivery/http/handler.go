// Package http (fitur report) = endpoint /api/v1/reports/weighbridge-daily dan /api/v1/reports/breakdown-daily.
package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/report/service"
	"simha/core-api/internal/shared/response"
)

// Handler = rekap laporan harian (F4).
type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/reports/weighbridge-daily", h.WeighbridgeDaily)
	api.GET("/reports/breakdown-daily", h.BreakdownDaily)
	api.GET("/reports/executive-hauling", h.ExecutiveHauling)
	api.GET("/reports/executive-emplacement", h.ExecutiveEmplacement)
}

// GET /api/v1/reports/executive-emplacement?project_id&date — per partner: produksi (Total TLS Out) harian/MTD/YTD + inventory (Sisa Cargo).
func (h *Handler) ExecutiveEmplacement(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	rows, err := h.svc.ExecutiveEmplacement(c.Request.Context(), response.ActorFrom(c), pid, c.Query("date"))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}

// GET /api/v1/reports/executive-hauling?project_id&date — per customer: daily_kg (tanggal itu), mtd_kg, ytd_kg (net kg).
func (h *Handler) ExecutiveHauling(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	rows, err := h.svc.ExecutiveHauling(c.Request.Context(), response.ActorFrom(c), pid, c.Query("date"))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}

func reportQuery(c *gin.Context) (service.ReportQuery, bool) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return service.ReportQuery{}, false
	}
	sid, ok := response.QueryUUID(c, "shift_id")
	if !ok {
		return service.ReportQuery{}, false
	}
	q := service.ReportQuery{ProjectID: pid, ShiftID: sid, From: c.Query("date_from"), To: c.Query("date_to")}
	if v := strings.TrimSpace(c.Query("equipment_code")); v != "" {
		q.Codes = strings.Split(strings.ToUpper(v), ",")
	}
	return q, true
}

// GET /api/v1/reports/weighbridge-daily?project_id&date_from&date_to&shift_id&equipment_code=A,B
// Per unit per tanggal: count = ritase, sum_kg = total net (payload = sum_kg/count/1000, produksi = sum_kg/1000).
func (h *Handler) WeighbridgeDaily(c *gin.Context) {
	q, ok := reportQuery(c)
	if !ok {
		return
	}
	rows, err := h.svc.WeighbridgeDaily(c.Request.Context(), response.ActorFrom(c), q)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}

// GET /api/v1/reports/breakdown-daily?… — per unit per tanggal: sum_hours = jam breakdown (bahan Breakdown & PA)
func (h *Handler) BreakdownDaily(c *gin.Context) {
	q, ok := reportQuery(c)
	if !ok {
		return
	}
	rows, err := h.svc.BreakdownDaily(c.Request.Context(), response.ActorFrom(c), q)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows)})
}
