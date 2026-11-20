package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"simha/core-api/internal/pagerule/domain"
	"simha/core-api/internal/pagerule/service"
	"simha/core-api/internal/shared/middleware"
	"simha/core-api/internal/shared/token"
)

type fakeRepo struct{ role []domain.Rule }

func (f *fakeRepo) ListProject(context.Context, uuid.UUID) ([]domain.Rule, error) { return nil, nil }
func (f *fakeRepo) ListFor(context.Context, uuid.UUID, string, uuid.UUID) ([]domain.Rule, []domain.Rule, error) {
	return f.role, nil, nil
}
func (f *fakeRepo) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error)        { return true, nil }
func (f *fakeRepo) Apply(context.Context, uuid.UUID, domain.SaveInput, uuid.UUID) error { return nil }

// panggil GET path sebagai pemanggil dengan claims tertentu; kembalikan kode HTTP
func call(t *testing.T, repo *fakeRepo, cl token.AccessClaims, path string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.ClaimsKey, &cl) })
	api := r.Group("/api/v1")
	api.Use(New(service.New(repo)).Guard())
	api.GET("/*any", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

func claims(role string, sa, internal bool) token.AccessClaims {
	return token.AccessClaims{
		ProjectID: uuid.NewString(), ProjectRole: role, SuperAdmin: sa, Internal: internal,
		RegisteredClaims: jwt.RegisteredClaims{Subject: uuid.NewString()},
	}
}

func TestGuard(t *testing.T) {
	open := &fakeRepo{}
	for name, c := range map[string]struct {
		repo *fakeRepo
		cl   token.AccessClaims
		path string
		want int
	}{
		"viewer ke data Record/Upload ditolak (bawaan)":    {open, claims("viewer", false, false), "/api/v1/weighbridge-tickets", 403},
		"viewer ke laporan executive ditolak":              {open, claims("viewer", false, false), "/api/v1/reports/executive-hauling", 403},
		"viewer ke rekap harian lolos (dipakai Dashboard)": {open, claims("viewer", false, false), "/api/v1/reports/weighbridge-daily", 200},
		"viewer ke endpoint tak dijaga lolos":              {open, claims("viewer", false, false), "/api/v1/shifts", 200},
		"dispatcher ke Record/Upload lolos (bawaan)":       {open, claims("dispatcher", false, false), "/api/v1/ritases", 200},
		"superadmin selalu lolos":                          {open, claims("", true, false), "/api/v1/ritases", 200},
		"token layanan cetak PDF lolos (role viewer)":      {open, claims("viewer", false, true), "/api/v1/reports/executive-hauling", 200},
		"dispatcher diblokir superadmin ditolak": {
			&fakeRepo{role: []domain.Rule{{SubjectType: "role", PageKey: "upload/ritase", Allowed: false}}},
			claims("dispatcher", false, false), "/api/v1/ritases", 403,
		},
		"viewer dibukakan superadmin lolos": {
			&fakeRepo{role: []domain.Rule{{SubjectType: "role", PageKey: "upload/ritase", Allowed: true}}},
			claims("viewer", false, false), "/api/v1/ritases", 200,
		},
	} {
		if got := call(t, c.repo, c.cl, c.path); got != c.want {
			t.Errorf("%s: kode %d, mau %d", name, got, c.want)
		}
	}
}
