// Package http (fitur importexcel) = endpoint /api/v1/imports/:type/template dan /api/v1/imports/:type.
package http

import (
	"errors"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/importexcel/service"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/excel"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Impor Excel (template sama dengan sistem lama): pratinjau ?dry_run=true, lalu simpan.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/imports/:type/template", h.Template)
	api.POST("/imports/:type", h.Import)
}

// GET /api/v1/imports/:type/template — file .xlsx sama dengan template sistem lama
func (h *Handler) Template(c *gin.Context) {
	typ := c.Param("type")
	b, err := h.svc.Template(typ)
	if response.WriteError(c, err) {
		return
	}
	name := "template_" + map[string]string{
		"operator-assignments": "setting_operator", "equipment-activations": "equipment_activation",
		"equipment-breakdowns": "breakdown_status", "weighbridge-tickets": "weighbridge",
		"stock-emplacements": "stock_emplacement", "yearly-plans": "plan_yearly", "monthly-plans": "plan_monthly",
		"time-sheets": "time_sheet", "ritases": "ritase", "work-hour-notifications": "wh_notification", "rain-slippery-logs": "rain_slippery",
	}[typ] + ".xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", b)
}

// POST /api/v1/imports/:type?dry_run=true&project_id= — multipart, field "file"
//
//	dry_run=true  → pratinjau: diperiksa, TIDAK disimpan (200)
//	tanpa dry_run → disimpan semua (200), atau ditolak semua bila ada baris salah (422 + daftar kesalahan)
func (h *Handler) Import(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, excel.MaxFileBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unggah file pada field \"file\" (maks. 5 MB)"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file tidak bisa dibuka"})
		return
	}
	defer f.Close()

	res, err := h.svc.Import(c.Request.Context(), response.ActorFrom(c), c.Param("type"), pid,
		filepath.Base(fh.Filename), f, c.Query("dry_run") == "true")
	if errors.Is(err, apperror.ErrImportHasErrors) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "data": res})
		return
	}
	if errors.Is(err, apperror.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "jenis impor tidak dikenal", "types": h.svc.Types()})
		return
	}
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}
