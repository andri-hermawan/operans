// Package http (fitur auth) = endpoint /auth/* (login, refresh, pindah project, logout, /me, ganti password)
// dan /.well-known/jwks.json. Baca request → panggil service → tulis JSON.
package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/core-api/internal/auth/service"
	"simha/core-api/internal/shared/middleware"
	"simha/core-api/internal/shared/response"
	"simha/core-api/internal/shared/token"
)

// Handler = endpoint /auth/* dan JWKS.
type Handler struct {
	svc    *service.Service
	signer *token.Signer
}

func New(svc *service.Service, signer *token.Signer) *Handler {
	return &Handler{svc: svc, signer: signer}
}

// RegisterPublic mendaftarkan route di LUAR /api/v1. Login & sesi tanpa access token (dijaga password /
// refresh token); hanya /auth/me dan /auth/change-password yang butuh access token.
func (h *Handler) RegisterPublic(r *gin.Engine) {
	r.GET("/.well-known/jwks.json", h.JWKS) // kunci publik untuk layanan lain

	requireAccess := middleware.RequireAccess(h.signer)
	authGroup := r.Group("/auth")
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)
	authGroup.POST("/switch-project", h.SwitchProject)
	authGroup.POST("/logout", h.Logout)
	authGroup.GET("/me", requireAccess, h.Me)
	authGroup.POST("/change-password", requireAccess, h.ChangePassword)
}

type loginRequest struct {
	Email     string     `json:"email"`
	Password  string     `json:"password"`
	ProjectID *uuid.UUID `json:"project_id"` // opsional
}

type refreshRequest struct {
	RefreshToken string     `json:"refresh_token"`
	ProjectID    *uuid.UUID `json:"project_id"` // diisi untuk pindah project
}

// POST /auth/login
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if !response.DecodeJSON(c, &req) {
		return
	}
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email dan password wajib diisi"})
		return
	}
	sess, err := h.svc.Login(c.Request.Context(), req.Email, req.Password, req.ProjectID, client(c))
	h.respond(c, sess, err)
}

// POST /auth/refresh — tukar refresh token dengan sesi baru (frontend memanggil ini sebelum access token habis).
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if !response.DecodeJSON(c, &req) || !requireRefresh(c, req.RefreshToken) {
		return
	}
	sess, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, req.ProjectID, client(c))
	h.respond(c, sess, err)
}

// POST /auth/switch-project — sama dengan refresh, tetapi project_id WAJIB.
func (h *Handler) SwitchProject(c *gin.Context) {
	var req refreshRequest
	if !response.DecodeJSON(c, &req) || !requireRefresh(c, req.RefreshToken) {
		return
	}
	if req.ProjectID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id wajib diisi"})
		return
	}
	sess, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, req.ProjectID, client(c))
	h.respond(c, sess, err)
}

// POST /auth/logout — selalu 204, juga bila token sudah tidak berlaku.
func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	if !response.DecodeJSON(c, &req) || !requireRefresh(c, req.RefreshToken) {
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		slog.Error("logout gagal", "request_id", c.GetString(middleware.RequestIDKey), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /auth/me — profil, daftar project & role. Butuh access token.
func (h *Handler) Me(c *gin.Context) {
	claims := c.MustGet(middleware.ClaimsKey).(*token.AccessClaims)
	userID, _ := uuid.Parse(claims.Subject)
	var active *uuid.UUID
	if id, err := uuid.Parse(claims.ProjectID); err == nil {
		active = &id
	}
	p, err := h.svc.Me(c.Request.Context(), userID, active)
	switch {
	case errors.Is(err, service.ErrInvalidRefresh):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "akun tidak aktif"})
	case err != nil:
		slog.Error("me gagal", "request_id", c.GetString(middleware.RequestIDKey), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	default:
		c.JSON(http.StatusOK, p)
	}
}

// POST /auth/change-password — {current_password, new_password}: semua sesi berakhir, login ulang.
func (h *Handler) ChangePassword(c *gin.Context) {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !response.DecodeJSON(c, &in) {
		return
	}
	err := h.svc.ChangePassword(c.Request.Context(), response.ActorFrom(c).UserID, in.CurrentPassword, in.NewPassword)
	if errors.Is(err, service.ErrWrongPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if response.WriteError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /.well-known/jwks.json — kunci PUBLIK untuk layanan lain (boleh dilihat siapa saja).
func (h *Handler) JWKS(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, h.signer.JWKS())
}

// respond menerjemahkan hasil service ke kode HTTP.
func (h *Handler) respond(c *gin.Context, sess *service.Session, err error) {
	var locked *service.LockedError
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store") // token tidak boleh disimpan cache mana pun
		c.JSON(http.StatusOK, sess)
	case errors.As(err, &locked):
		c.Header("Retry-After", retryAfter(locked.Until))
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInvalidRefresh),
		errors.Is(err, service.ErrRefreshReused):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNoProject), errors.Is(err, service.ErrProjectForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		slog.Error("auth gagal", "request_id", c.GetString(middleware.RequestIDKey), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
}

func requireRefresh(c *gin.Context, refreshToken string) bool {
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token wajib diisi"})
		return false
	}
	return true
}

func client(c *gin.Context) service.Client {
	return service.Client{UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
}

// retryAfter = jumlah detik sampai akun boleh mencoba lagi (header standar Retry-After).
func retryAfter(until time.Time) string {
	return strconv.Itoa(max(int(time.Until(until).Seconds())+1, 1))
}
