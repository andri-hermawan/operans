// Package http (fitur reportschedule) = endpoint /api/v1/report-schedules (jadwal, log, kirim sekarang, pratinjau PDF).
package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/reportschedule/domain"
	"simha/core-api/internal/reportschedule/service"
	"simha/core-api/internal/shared/response"
)

// Handler = jadwal kirim laporan (F5).
type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// /report-schedules/types (statis) dan /report-schedules/:id (parameter) boleh berdampingan di gin.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/report-schedules/types", h.Types)
	api.GET("/report-schedules", h.List)
	api.POST("/report-schedules", h.Save)
	api.PATCH("/report-schedules/:id", h.Save)
	api.DELETE("/report-schedules/:id", h.Delete)
	api.GET("/report-schedules/:id/logs", h.Logs)
	api.POST("/report-schedules/:id/send-now", h.SendNow)
	api.GET("/report-schedules/:id/preview", h.Preview)
}

// GET /api/v1/report-schedules/types
func (h *Handler) Types(c *gin.Context) {
	out := []gin.H{}
	for code, t := range domain.ReportTypes {
		out = append(out, gin.H{"code": code, "label": t.Label})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// GET /api/v1/report-schedules?project_id=
func (h *Handler) List(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/report-schedules · PATCH /api/v1/report-schedules/:id
func (h *Handler) Save(c *gin.Context) {
	id, ok := response.OptionalPathUUID(c)
	if !ok {
		return
	}
	var in domain.ReportScheduleInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	s, err := h.svc.Save(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.SavedStatus(id), gin.H{"data": s})
}

// DELETE /api/v1/report-schedules/:id
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

// GET /api/v1/report-schedules/:id/logs
func (h *Handler) Logs(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.Logs(c.Request.Context(), response.ActorFrom(c), id)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/report-schedules/:id/send-now — hasil (berhasil/gagal) juga tercatat di log
func (h *Handler) SendNow(c *gin.Context) {
	extendWrite(c)
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	l, err := h.svc.SendNow(c.Request.Context(), response.ActorFrom(c), id)
	if l != nil && err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "data": l})
		return
	}
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": l})
}

// GET /api/v1/report-schedules/:id/preview?date= — PDF tanpa mengirim email
func (h *Handler) Preview(c *gin.Context) {
	extendWrite(c)
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	pdf, name, err := h.svc.Preview(c.Request.Context(), response.ActorFrom(c), id, c.Query("date"))
	if err != nil && pdf == nil && name != "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "buat PDF gagal: " + err.Error()})
		return
	}
	if response.WriteError(c, err) {
		return
	}
	c.Header("Content-Disposition", `inline; filename="`+name+`"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// extendWrite: membuat PDF bisa lebih lama dari batas tulis server (30 detik).
func extendWrite(c *gin.Context) {
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(3 * time.Minute))
}
