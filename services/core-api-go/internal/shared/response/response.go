// Package response = bantuan untuk handler HTTP semua fitur: membaca isian (JSON, UUID, angka, waktu)
// dari request, mengambil Actor dari token, dan menulis kesalahan dengan kode HTTP yang seragam.
package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/middleware"
	"simha/core-api/internal/shared/token"
)

const maxJSONBody = 8 * 1024 // permintaan login/refresh kecil (batas yang sama dipakai semua isian JSON)

// ActorFrom = pemanggil, dari access token yang sudah diperiksa RequireAccess.
// Catatan: role di token berlaku sampai token habis (maks. 15 menit) walau role diubah di database.
func ActorFrom(c *gin.Context) access.Actor {
	cl := c.MustGet(middleware.ClaimsKey).(*token.AccessClaims)
	a := access.Actor{Role: cl.ProjectRole, SuperAdmin: cl.SuperAdmin}
	a.UserID, _ = uuid.Parse(cl.Subject)
	if pid, err := uuid.Parse(cl.ProjectID); err == nil {
		a.ProjectID = &pid
	}
	return a
}

// WriteError menerjemahkan kesalahan service ke kode HTTP. true = sudah dibalas (pemanggil berhenti).
func WriteError(c *gin.Context, err error) bool {
	var ve *apperror.ValidationError
	switch {
	case err == nil:
		return false
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Msg})
	case errors.Is(err, apperror.ErrForbidden), errors.Is(err, apperror.ErrSelf):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, apperror.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, apperror.ErrEmailExists), errors.Is(err, apperror.ErrCodeExists), errors.Is(err, apperror.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, apperror.ErrInvalidRefresh):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "akun tidak aktif"})
	default:
		slog.Error("request gagal", "request_id", c.GetString(middleware.RequestIDKey), "path", c.FullPath(), "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	}
	return true
}

// DecodeJSON membaca isi JSON ke dst. Field yang tidak dikenal ditolak. false = sudah dibalas 400.
func DecodeJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBody)
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON tidak valid: " + err.Error()})
		return false
	}
	return true
}

// QueryUUID: ?name= (kosong → nil). false = sudah dibalas 400.
func QueryUUID(c *gin.Context, name string) (*uuid.UUID, bool) {
	v := c.Query(name)
	if v == "" {
		return nil, true
	}
	id, err := uuid.Parse(v)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " bukan UUID"})
		return nil, false
	}
	return &id, true
}

// PathUUID: /…/:name harus UUID. false = sudah dibalas 400.
func PathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " bukan UUID"})
		return uuid.Nil, false
	}
	return id, true
}

// PathInt: /…/:name harus angka > 0. false = sudah dibalas 400.
func PathInt(c *gin.Context, name string) (int64, bool) {
	n, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || n <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " harus angka"})
		return 0, false
	}
	return n, true
}

// OptionalPathUUID: tanpa :id / :projectId (POST) → nil.
func OptionalPathUUID(c *gin.Context) (*uuid.UUID, bool) {
	name := "id"
	if c.Param("projectId") != "" {
		name = "projectId"
	}
	if c.Param(name) == "" {
		return nil, true
	}
	id, ok := PathUUID(c, name)
	return &id, ok
}

// QueryTime: ?name= format RFC3339 (kosong → nil). false = sudah dibalas 400.
func QueryTime(c *gin.Context, name string) (*time.Time, bool) {
	v := c.Query(name)
	if v == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " harus format RFC3339, mis. 2026-10-23T08:00:00+07:00"})
		return nil, false
	}
	return &t, true
}

// DailyFilter membaca filter data harian: ?project_id=&date_from=&date_to=&shift_id=&equipment_code=
func DailyFilter(c *gin.Context) (dailyinput.DailyFilter, bool) {
	pid, ok := QueryUUID(c, "project_id")
	if !ok {
		return dailyinput.DailyFilter{}, false
	}
	sid, ok := QueryUUID(c, "shift_id")
	if !ok {
		return dailyinput.DailyFilter{}, false
	}
	return dailyinput.DailyFilter{ProjectID: pid, ShiftID: sid, DateFrom: c.Query("date_from"), DateTo: c.Query("date_to"),
		EquipmentCode: c.Query("equipment_code")}, true
}

// SavedStatus: tambah (id nil) → 201, ubah → 200.
func SavedStatus(id *uuid.UUID) int {
	if id == nil {
		return http.StatusCreated
	}
	return http.StatusOK
}

// CreatedStatus: baris baru → 201, diperbarui → 200.
func CreatedStatus(created bool) int {
	if created {
		return http.StatusCreated
	}
	return http.StatusOK
}

// NonNil: daftar kosong dikirim sebagai [] (bukan null).
func NonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
