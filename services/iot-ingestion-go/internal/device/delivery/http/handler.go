// Package http = pintu HTTP fitur Device: baca request → panggil service → tulis JSON.
// Daftar route fitur ini ada di Register.
package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/device/service"
	"simha/iot-ingestion/internal/shared/middleware"
	"simha/iot-ingestion/internal/shared/response"
)

// Kelola alat GPS (System Data → Device). Lihat: admin project & superadmin. Ubah: sama.
// Alat tanpa project (baru didaftarkan pabrik) hanya bisa diatur superadmin.

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Register mendaftarkan route fitur Device (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/devices/:deviceId/status", h.DeviceStatus)
	api.GET("/devices", h.Devices)
	api.POST("/devices", h.SaveDevice)
	api.PATCH("/devices/:deviceId", h.SaveDevice)
	api.POST("/devices/:deviceId/api-key", h.RotateDeviceKey)
}

var imeiPattern = regexp.MustCompile(`^[0-9]{15}$`)

func canManageDevices(p middleware.Principal) bool { return p.SuperAdmin || p.ProjectRole == "admin" }

// GET /api/v1/devices
func (h *Handler) Devices(c *gin.Context) {
	p := response.Principal(c)
	if !canManageDevices(p) {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya admin project / superadmin"})
		return
	}
	list, err := h.svc.List(c.Request.Context(), p.ProjectScope())
	if response.InternalErr(c, err, "daftar alat gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "count": len(list)})
}

// POST /api/v1/devices · PATCH /api/v1/devices/:deviceId
func (h *Handler) SaveDevice(c *gin.Context) {
	p := response.Principal(c)
	if !canManageDevices(p) {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya admin project / superadmin"})
		return
	}
	var in domain.DeviceInput
	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON tidak valid: " + err.Error()})
		return
	}
	trim := func(s *string) {
		if s != nil {
			*s = strings.TrimSpace(*s)
		}
	}
	for _, s := range []*string{in.IMEI, in.Name, in.Model, in.FuelSource, in.SimNumber, in.ProviderName} {
		trim(s)
	}
	if in.FuelSource != nil && *in.FuelSource != "" && *in.FuelSource != "escort_ble" && *in.FuelSource != "ecu" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fuel_source: escort_ble, ecu, atau kosong"})
		return
	}

	ctx := c.Request.Context()
	var id *uuid.UUID
	var cur *domain.DeviceRow
	if raw := c.Param("deviceId"); raw != "" {
		v, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId bukan UUID"})
			return
		}
		id = &v
		var gerr error
		cur, gerr = h.svc.Get(ctx, v)
		if response.InternalErr(c, gerr, "baca alat gagal") {
			return
		}
		if cur == nil || !p.SuperAdmin && (cur.ProjectID == nil || *cur.ProjectID != p.ProjectID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alat tidak ditemukan"})
			return
		}
		if in.IMEI != nil && *in.IMEI != cur.IMEI {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IMEI tidak bisa diubah; daftarkan sebagai alat baru"})
			return
		}
		in.IMEI, in.ProjectID = nil, nil // project mengikuti equipment
	} else {
		if in.IMEI == nil || !imeiPattern.MatchString(*in.IMEI) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "IMEI wajib 15 digit angka"})
			return
		}
		if !p.SuperAdmin || in.ProjectID == nil {
			pid := p.ProjectID
			in.ProjectID = &pid
		}
	}

	// equipment harus satu project dengan alat, dan satu unit hanya boleh punya satu alat aktif
	if in.EquipmentID != nil && !in.ClearUnit {
		ep, err := h.svc.EquipmentProject(ctx, *in.EquipmentID)
		if response.InternalErr(c, err, "cek equipment gagal") {
			return
		}
		devProject := in.ProjectID
		if cur != nil {
			devProject = cur.ProjectID
		}
		if ep == nil || devProject == nil || *ep != *devProject {
			c.JSON(http.StatusBadRequest, gin.H{"error": "equipment tidak ditemukan di project alat ini"})
			return
		}
		if other, err := h.svc.DeviceOnEquipment(ctx, *in.EquipmentID, id); response.InternalErr(c, err, "cek alat gagal") {
			return
		} else if other != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "equipment ini sudah memakai alat aktif IMEI " + other})
			return
		}
	}

	if id == nil {
		newID, err := h.svc.Insert(ctx, in)
		if errors.Is(err, domain.ErrIMEIExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if response.InternalErr(c, err, "tambah alat gagal") {
			return
		}
		id = &newID
	} else if response.InternalErr(c, h.svc.Update(ctx, *id, in), "ubah alat gagal") {
		return
	}
	d, err := h.svc.Get(ctx, *id)
	if response.InternalErr(c, err, "baca alat gagal") {
		return
	}
	status := http.StatusOK
	if cur == nil {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"data": d})
}

// POST /api/v1/devices/:deviceId/api-key — buat API key baru (alat HTTP/simulator). Key hanya ditampilkan SEKALI;
// yang disimpan hanya hash-nya. Key lama langsung tidak berlaku.
func (h *Handler) RotateDeviceKey(c *gin.Context) {
	p := response.Principal(c)
	if !canManageDevices(p) {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya admin project / superadmin"})
		return
	}
	id, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId bukan UUID"})
		return
	}
	cur, err := h.svc.Get(c.Request.Context(), id)
	if response.InternalErr(c, err, "baca alat gagal") {
		return
	}
	if cur == nil || !p.SuperAdmin && (cur.ProjectID == nil || *cur.ProjectID != p.ProjectID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "alat tidak ditemukan"})
		return
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); response.InternalErr(c, err, "buat key gagal") {
		return
	}
	key := "simha_" + base64.RawURLEncoding.EncodeToString(buf)
	if response.InternalErr(c, h.svc.SetAPIKey(c.Request.Context(), id, key), "simpan key gagal") {
		return
	}
	slog.Info("API key alat diganti", "request_id", response.RequestID(c), "device_id", id, "by", p.UserID) // JANGAN log key-nya
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"api_key": key})
}

// DeviceStatus: kapan alat terakhir mengirim data.
func (h *Handler) DeviceStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId bukan UUID"})
		return
	}
	eventTime, receivedAt, err := h.svc.LastEvent(c.Request.Context(), response.Principal(c).ProjectScope(), id)
	if err != nil {
		slog.Error("status alat gagal", "request_id", response.RequestID(c), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": id, "last_event_time": eventTime, "last_received_at": receivedAt})
}
