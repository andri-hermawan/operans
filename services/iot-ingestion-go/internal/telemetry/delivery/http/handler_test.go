package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/shared/middleware"
)

// Parameter History yang salah harus ditolak SEBELUM menyentuh database (service sengaja nil).
func TestHistoryParameterSalah(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{maxRange: 24 * time.Hour, maxRows: 1000}
	r := gin.New()
	r.GET("/equipment/:equipmentId/history",
		func(c *gin.Context) {
			c.Set(middleware.CtxPrincipal, middleware.Principal{ProjectID: uuid.New(), ProjectRole: middleware.RoleViewer})
		},
		h.History)

	eq := uuid.NewString()
	tests := map[string]string{
		"bukan UUID":           "/equipment/HD-01/history?from=2026-10-22T00:00:00Z&to=2026-10-22T01:00:00Z",
		"tanpa from/to":        "/equipment/" + eq + "/history",
		"to sebelum from":      "/equipment/" + eq + "/history?from=2026-10-22T05:00:00Z&to=2026-10-22T01:00:00Z",
		"rentang 48 jam":       "/equipment/" + eq + "/history?from=2026-10-20T00:00:00Z&to=2026-10-22T00:00:00Z",
		"format tanggal salah": "/equipment/" + eq + "/history?from=22-10-2026&to=23-10-2026",
	}
	for name, url := range tests {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, seharusnya 400", w.Code)
			}
		})
	}
}
