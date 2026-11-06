// Package http (fitur project) = endpoint /api/v1/projects: baca request → panggil service → tulis JSON.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/project/domain"
	"simha/core-api/internal/project/service"
	"simha/core-api/internal/shared/response"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Catatan: /projects/:projectId/members milik fitur user — nama parameter harus sama (:projectId).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/projects", h.List)
	api.POST("/projects", h.Save)
	api.PATCH("/projects/:projectId", h.Save)
}

// GET /api/v1/projects — superadmin: semua; admin: project aktifnya
func (h *Handler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), response.ActorFrom(c))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/projects · PATCH /api/v1/projects/:id (superadmin)
func (h *Handler) Save(c *gin.Context) {
	id, ok := response.OptionalPathUUID(c)
	if !ok {
		return
	}
	var in domain.ProjectInput
	if !response.DecodeJSON(c, &in) {
		return
	}
	p, err := h.svc.Save(c.Request.Context(), response.ActorFrom(c), id, in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(response.SavedStatus(id), gin.H{"data": p})
}
