package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/repository"
	"simha/iot-ingestion/internal/ingest/service"
)

// memStore = database palsu di memori untuk service.Ingest.
type memStore struct {
	saved  map[string]bool
	dbDown bool
}

func (m *memStore) InsertEvent(_ context.Context, _ *devicedomain.Device, e *domain.Event) (repository.Saved, bool, error) {
	if m.dbDown {
		return repository.Saved{}, false, errors.New("database mati")
	}
	if m.saved[e.EventID] {
		return repository.Saved{}, false, nil
	}
	m.saved[e.EventID] = true
	return repository.Saved{ID: int64(len(m.saved)), ReceivedAt: time.Now()}, true, nil
}

func TestPostEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dev := &devicedomain.Device{ID: uuid.New(), IMEI: "000000000000001"}
	t0 := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)

	newRouter := func(store *memStore) *gin.Engine {
		h := &Handler{ingest: service.NewIngest(store, nil, nil)}
		r := gin.New()
		// Pengganti DeviceAuth: anggap API key sudah benar dan alatnya dev.
		r.POST("/events", func(c *gin.Context) { c.Set(ctxDevice, dev) }, h.PostEvent)
		return r
	}
	send := func(r *gin.Engine, body string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body)))
		return w.Code
	}

	ok := `{"event_id":"e1","event_time":"` + t0 + `","latitude":-3.47,"longitude":103.85}`

	t.Run("baru 201, kirim ulang 200", func(t *testing.T) {
		r := newRouter(&memStore{saved: map[string]bool{}})
		if c := send(r, ok); c != http.StatusCreated {
			t.Fatalf("kirim pertama = %d", c)
		}
		if c := send(r, ok); c != http.StatusOK {
			t.Fatalf("kirim ulang = %d", c)
		}
	})

	tests := []struct {
		name  string
		body  string
		store *memStore
		want  int
	}{
		{"JSON rusak", `{"event_id":`, &memStore{saved: map[string]bool{}}, http.StatusBadRequest},
		{"field salah ketik", `{"event_id":"e2","event_time":"` + t0 + `","fuel_levle_raw":1}`, &memStore{saved: map[string]bool{}}, http.StatusBadRequest},
		{"latitude 999", `{"event_id":"e3","event_time":"` + t0 + `","latitude":999,"longitude":1}`, &memStore{saved: map[string]bool{}}, http.StatusUnprocessableEntity},
		{"database mati", ok, &memStore{dbDown: true}, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if c := send(newRouter(tt.store), tt.body); c != tt.want {
				t.Fatalf("status = %d, seharusnya %d", c, tt.want)
			}
		})
	}
}

func (m *memStore) InsertGeofenceEvent(context.Context, *domain.GeofenceEvent) error { return nil }

func (m *memStore) InsertAlert(context.Context, *domain.Alert) error         { return nil }
func (m *memStore) InsertFuelEvent(context.Context, *domain.FuelEvent) error { return nil }

func (m *memStore) CloseAlert(context.Context, uuid.UUID, string, time.Time) (*domain.Alert, error) {
	return nil, nil
}

func (m *memStore) ExtendAlert(context.Context, uuid.UUID, string, time.Time, time.Time) (*domain.Alert, error) {
	return nil, nil
}
