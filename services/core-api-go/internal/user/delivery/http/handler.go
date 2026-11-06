// Package http (fitur user) = endpoint kelola user (/api/v1/users) & anggota project
// (/api/v1/projects/:projectId/members). Baca request → panggil service → tulis JSON.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/user/domain"
	"simha/core-api/internal/user/service"
)

// Handler = kelola user & keanggotaan project (B3).
type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur ini di grup /api/v1 (sudah dijaga RequireAccess).
// Ganti password sendiri (/auth/change-password) ada di fitur auth.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/users", h.List)
	api.POST("/users", h.Create)
	api.GET("/users/:id", h.Get)
	api.PATCH("/users/:id", h.Update)
	api.POST("/users/:id/reset-password", h.ResetPassword)
	api.PUT("/projects/:projectId/members", h.SetMember)
	api.DELETE("/projects/:projectId/members/:userId", h.RemoveMember)
}

// GET /api/v1/users?q=&project_id=
func (h *Handler) List(c *gin.Context) {
	var filter *uuid.UUID
	if v := c.Query("project_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "project_id bukan UUID"})
			return
		}
		filter = &id
	}
	users, err := h.svc.List(c.Request.Context(), response.ActorFrom(c), filter, c.Query("q"))
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "count": len(users)})
}

// GET /api/v1/users/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	u, err := h.svc.Get(c.Request.Context(), response.ActorFrom(c), id)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}

// POST /api/v1/users
func (h *Handler) Create(c *gin.Context) {
	var in service.NewUser
	if !response.DecodeJSON(c, &in) {
		return
	}
	u, err := h.svc.Create(c.Request.Context(), response.ActorFrom(c), in)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": u})
}

// PATCH /api/v1/users/:id — {name?, phone_number?, is_active?, is_superadmin?}
func (h *Handler) Update(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	var upd domain.UserUpdate
	if !response.DecodeJSON(c, &upd) {
		return
	}
	u, err := h.svc.Update(c.Request.Context(), response.ActorFrom(c), id, upd)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}

// POST /api/v1/users/:id/reset-password — {new_password} (superadmin)
func (h *Handler) ResetPassword(c *gin.Context) {
	id, ok := response.PathUUID(c, "id")
	if !ok {
		return
	}
	var in struct {
		NewPassword string `json:"new_password"`
	}
	if !response.DecodeJSON(c, &in) {
		return
	}
	if response.WriteError(c, h.svc.ResetPassword(c.Request.Context(), response.ActorFrom(c), id, in.NewPassword)) {
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /api/v1/projects/:projectId/members — {email, role}: tambah anggota / ubah role
func (h *Handler) SetMember(c *gin.Context) {
	pid, ok := response.PathUUID(c, "projectId")
	if !ok {
		return
	}
	var in struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !response.DecodeJSON(c, &in) {
		return
	}
	u, err := h.svc.SetMember(c.Request.Context(), response.ActorFrom(c), pid, in.Email, in.Role)
	if response.WriteError(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}

// DELETE /api/v1/projects/:projectId/members/:userId
func (h *Handler) RemoveMember(c *gin.Context) {
	pid, ok := response.PathUUID(c, "projectId")
	if !ok {
		return
	}
	uid, ok := response.PathUUID(c, "userId")
	if !ok {
		return
	}
	if response.WriteError(c, h.svc.RemoveMember(c.Request.Context(), response.ActorFrom(c), pid, uid)) {
		return
	}
	c.Status(http.StatusNoContent)
}
