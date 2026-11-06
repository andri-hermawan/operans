// Package http = pintu HTTP fitur livestate: kondisi terakhir equipment untuk peta live & detail unit.
// Dibaca dari memori (service.Cache), jadi tidak ada query database.
package http

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/shared/response"
)

// LiveReader = pembaca kondisi terakhir equipment dari MEMORI (dipenuhi *service.Cache fitur ini).
type LiveReader interface {
	List(projectID *uuid.UUID) []domain.LiveState
	Get(equipmentID uuid.UUID) (domain.LiveState, bool)
}

type Handler struct {
	live LiveReader
}

func New(live LiveReader) *Handler { return &Handler{live: live} }

// Register mendaftarkan route fitur livestate (api = grup /api/v1 yang sudah dijaga token pengguna).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/equipment-live-states", h.LiveStates)            // semua equipment project (peta)
	api.GET("/equipment/:equipmentId/live-state", h.LiveState) // satu equipment (detail)
}

// LiveStates: kondisi terakhir SEMUA equipment di project pemanggil (superadmin: semua project).
// Data awal halaman peta; setelah itu perubahan dikirim lewat WebSocket (Fase A4).
// Dibaca dari memori, jadi tidak ada query database.
func (h *Handler) LiveStates(c *gin.Context) {
	states := h.live.List(response.Principal(c).ProjectScope())
	sort.Slice(states, func(i, j int) bool { // urutan tetap agar tampilan tidak "melompat"
		return states[i].EquipmentID.String() < states[j].EquipmentID.String()
	})
	c.JSON(http.StatusOK, gin.H{
		"data":  states,
		"count": len(states),
		"as_of": time.Now().UTC(),
	})
}

// LiveState: kondisi terakhir SATU equipment (halaman detail unit).
func (h *Handler) LiveState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("equipmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "equipmentId bukan UUID"})
		return
	}
	s, ok := h.live.Get(id)
	scope := response.Principal(c).ProjectScope()
	if !ok || (scope != nil && (s.ProjectID == nil || *s.ProjectID != *scope)) {
		// Sama untuk "belum ada" dan "milik project lain": tidak membocorkan keberadaan data.
		c.JSON(http.StatusNotFound, gin.H{"error": "belum ada data untuk equipment ini"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
}
