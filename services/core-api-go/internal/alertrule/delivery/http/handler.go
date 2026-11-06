// Package http (fitur alertrule) = endpoint /api/v1/alert-rules (+ /effective).
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/alertrule/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/alert-rules", h.List)
	api.GET("/alert-rules/effective", h.Effective)
	api.POST("/alert-rules", h.Create)
	api.PUT("/alert-rules/:id", h.Replace)
	api.DELETE("/alert-rules/:id", h.Delete)
}

// GET /api/v1/alert-rules?project_id=
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

// POST /api/v1/alert-rules (admin)
func (h *Handler) Create(c *gin.Context) {
	var in service.AlertRuleInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	r, warnings, err := h.svc.Create(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": r, "warnings": response.NonNil(warnings)})
}

// PUT /api/v1/alert-rules/:id — ganti seluruh isi aturan (admin)
func (h *Handler) Replace(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	var in service.AlertRuleInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	r, warnings, err := h.svc.Replace(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": r, "warnings": response.NonNil(warnings)})
}

// DELETE /api/v1/alert-rules/:id (admin)
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

// GET /api/v1/alert-rules/effective?project_id=&zone_name= — ambang yang berlaku + penerima
func (h *Handler) Effective(c *gin.Context) {
	pid, ok := response.QueryUUID(c, "project_id")
	if !ok {
		return
	}
	list, err := h.svc.Effective(c.Request.Context(), response.ActorFrom(c), pid, c.Query("zone_name"))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}
