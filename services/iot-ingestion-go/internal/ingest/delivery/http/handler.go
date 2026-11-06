// Package http = pintu HTTP fitur ingest: alat/simulator mengirim SATU event JSON (POST /api/v1/ingest/events)
// dengan header X-API-Key (diperiksa DeviceAuth di auth.go). Pintu TCP Teltonika ada di delivery/tcp;
// keduanya memakai service yang SAMA.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/service"
	"simha/iot-ingestion/internal/shared/response"
)

const maxBodyBytes = 64 * 1024 // satu event jauh di bawah 64 KB

type Handler struct {
	ingest  *service.Ingest
	devices DeviceFinder
}

func New(ingest *service.Ingest, devices DeviceFinder) *Handler {
	return &Handler{ingest: ingest, devices: devices}
}

// Register mendaftarkan route fitur ingest. api = grup /api/v1 TANPA token pengguna:
// alat/simulator mengirim data → butuh X-API-Key.
func (h *Handler) Register(api *gin.RouterGroup) {
	api.POST("/ingest/events", DeviceAuth(h.devices), h.PostEvent)
}

// PostEvent: alat/simulator mengirim SATU event (JSON) dengan header X-API-Key.
func (h *Handler) PostEvent(c *gin.Context) {
	dev := c.MustGet(ctxDevice).(*devicedomain.Device)

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields() // salah ketik nama field (mis. "fuel_levle_raw") langsung ketahuan

	var e domain.Event
	if err := dec.Decode(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON tidak valid: " + err.Error()})
		return
	}

	res, err := h.ingest.Handle(c.Request.Context(), dev, &e)
	switch {
	case service.IsInvalid(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case err != nil:
		slog.Error("simpan event gagal", "request_id", response.RequestID(c), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
	case res.Duplicate:
		c.JSON(http.StatusOK, gin.H{"status": "duplicate", "event_id": e.EventID}) // aman dikirim ulang
	default:
		c.JSON(http.StatusCreated, gin.H{"status": "stored", "event_id": e.EventID})
	}
}
