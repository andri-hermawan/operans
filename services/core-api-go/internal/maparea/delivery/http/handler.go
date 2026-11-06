// Package http (fitur maparea) = endpoint peta /api/v1/geo/zones (+ lookup) dan /api/v1/geo/imports.
package http

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/core-api/internal/maparea/domain"
	"simha/core-api/internal/maparea/service"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/response"
)

// Handler = peta project (B6, ADR 0003).
type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Upload GeoJSON berversi — pratinjau ?dry_run=true, lalu simpan (admin project).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/geo/zones", h.Zones)
	api.GET("/geo/zones/lookup", h.Lookup)
	api.GET("/geo/imports", h.Imports)
	api.GET("/geo/imports/:id/file", h.ImportFile)
	api.POST("/geo/imports", h.Import)
}

// GET /api/v1/geo/zones?project_id=&category=&at= — GeoJSON FeatureCollection (application/geo+json).
// at (RFC3339) = peta yang berlaku pada waktu itu; kosong = sekarang.
func (h *Handler) Zones(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	at, ok := response.QueryTime(c, "at")
	if !ok {
		return
	}
	b, err := h.svc.Zones(c.Request.Context(), response.ActorFrom(c), pid, c.Query("category"), at)
	if response.WriteError(c, err) {
		return
	}
	c.Data(http.StatusOK, "application/geo+json", b)
}

// GET /api/v1/geo/zones/lookup?lat=&lon=&at=&project_id= — zona tempat titik berada ({"data": null} bila di luar).
func (h *Handler) Lookup(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	at, ok := response.QueryTime(c, "at")
	if !ok {
		return
	}
	lat, err1 := strconv.ParseFloat(c.Query("lat"), 64)
	lon, err2 := strconv.ParseFloat(c.Query("lon"), 64)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat dan lon wajib berupa angka derajat"})
		return
	}
	z, err := h.svc.Lookup(c.Request.Context(), response.ActorFrom(c), pid, lat, lon, at)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": z})
}

// GET /api/v1/geo/imports?project_id= — riwayat upload peta
func (h *Handler) Imports(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.Imports(c.Request.Context(), response.ActorFrom(c), pid)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GET /api/v1/geo/imports/:id/file — unduh file GeoJSON asli
func (h *Handler) ImportFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id bukan UUID"})
		return
	}
	name, raw, err := h.svc.ImportFile(c.Request.Context(), response.ActorFrom(c), id)
	if response.WriteError(c, err) {
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filepath.Base(name)+`"`)
	c.Data(http.StatusOK, "application/geo+json", raw)
}

// POST /api/v1/geo/imports?dry_run=true&project_id= — multipart: field "file" (+ "note" opsional)
//
//	dry_run=true  → pratinjau: diperiksa (termasuk geometri oleh PostGIS), TIDAK disimpan
//	tanpa dry_run → peta lama ditutup & peta baru berlaku sekarang, atau ditolak semua (422)
func (h *Handler) Import(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, domain.MaxFileBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unggah file pada field \"file\" (maks. 10 MB)"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file tidak bisa dibuka"})
		return
	}
	defer f.Close()
	var note *string
	if v, has := c.GetPostForm("note"); has {
		note = &v
	}

	res, err := h.svc.Import(c.Request.Context(), response.ActorFrom(c), pid, filepath.Base(fh.Filename), note, f,
		c.Query("dry_run") == "true")
	if errors.Is(err, apperror.ErrImportHasErrors) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "data": res})
		return
	}
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}
