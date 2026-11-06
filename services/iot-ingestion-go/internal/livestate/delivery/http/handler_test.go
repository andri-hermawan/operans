package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/shared/middleware"
)

// memLive = pengganti service.Cache untuk test.
type memLive map[uuid.UUID]domain.LiveState

func (m memLive) List(projectID *uuid.UUID) []domain.LiveState {
	out := []domain.LiveState{}
	for _, s := range m {
		if projectID == nil || (s.ProjectID != nil && *s.ProjectID == *projectID) {
			out = append(out, s)
		}
	}
	return out
}

func (m memLive) Get(id uuid.UUID) (domain.LiveState, bool) {
	s, ok := m[id]
	return s, ok
}

func TestLiveStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectA, projectB := uuid.New(), uuid.New()
	eqA1, eqA2, eqB := uuid.New(), uuid.New(), uuid.New()
	live := memLive{
		eqA1: {EquipmentID: eqA1, ProjectID: &projectA},
		eqA2: {EquipmentID: eqA2, ProjectID: &projectA},
		eqB:  {EquipmentID: eqB, ProjectID: &projectB},
	}
	h := &Handler{live: live}

	call := func(p middleware.Principal, url string) (*httptest.ResponseRecorder, map[string]json.RawMessage) {
		r := gin.New()
		setP := func(c *gin.Context) { c.Set(middleware.CtxPrincipal, p) }
		r.GET("/equipment-live-states", setP, h.LiveStates)
		r.GET("/equipment/:equipmentId/live-state", setP, h.LiveState)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		var body map[string]json.RawMessage
		json.Unmarshal(w.Body.Bytes(), &body)
		return w, body
	}
	viewerA := middleware.Principal{ProjectID: projectA, ProjectRole: middleware.RoleViewer}
	superadmin := middleware.Principal{SuperAdmin: true}

	t.Run("viewer project A hanya melihat equipment project A", func(t *testing.T) {
		w, body := call(viewerA, "/equipment-live-states")
		if w.Code != http.StatusOK || string(body["count"]) != "2" {
			t.Fatalf("status=%d count=%s", w.Code, body["count"])
		}
	})

	t.Run("superadmin melihat semua project", func(t *testing.T) {
		_, body := call(superadmin, "/equipment-live-states")
		if string(body["count"]) != "3" {
			t.Fatalf("count=%s", body["count"])
		}
	})

	t.Run("detail equipment sendiri 200, project lain 404, tidak ada 404, bukan UUID 400", func(t *testing.T) {
		cases := map[string]int{
			"/equipment/" + eqA1.String() + "/live-state":    http.StatusOK,
			"/equipment/" + eqB.String() + "/live-state":     http.StatusNotFound,
			"/equipment/" + uuid.NewString() + "/live-state": http.StatusNotFound,
			"/equipment/HD-01/live-state":                    http.StatusBadRequest,
		}
		for url, want := range cases {
			if w, _ := call(viewerA, url); w.Code != want {
				t.Errorf("%s → %d, seharusnya %d", url, w.Code, want)
			}
		}
	})
}
