package response

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simha/iot-ingestion/internal/shared/database"
)

// Endpoint baca halaman pemantauan (F3/F4). Semua: ?date=YYYY-MM-DD (tanggal operasional) &shift=SHIFT_1
// &equipment_id=… — dibatasi project pemanggil.

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// OptStr = nilai query `name` yang sudah dirapikan; nil bila kosong.
func OptStr(c *gin.Context, name string) *string {
	if v := strings.TrimSpace(c.Query(name)); v != "" {
		return &v
	}
	return nil
}

// ValidDate: v berformat YYYY-MM-DD dan tanggalnya ada.
func ValidDate(v string) bool {
	if !dateRe.MatchString(v) {
		return false
	}
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}

// ReadSlot membaca filter standar; false = sudah dijawab 400.
func ReadSlot(c *gin.Context) (database.Slot, bool) {
	s := database.Slot{ProjectID: Principal(c).ProjectScope(), Date: c.Query("date"), ShiftCode: OptStr(c, "shift")}
	if !ValidDate(s.Date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date wajib format YYYY-MM-DD (tanggal operasional)"})
		return s, false
	}
	if v := c.Query("equipment_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "equipment_id bukan UUID"})
			return s, false
		}
		s.EquipmentID = &id
	}
	return s, true
}

// ReadRange membaca ?date (atau date_from & date_to, maks. 93 hari)&shift&type&equipment_id&limit.
func ReadRange(c *gin.Context) (database.RangeFilter, bool) {
	f := database.RangeFilter{ProjectID: Principal(c).ProjectScope(), From: c.Query("date_from"), To: c.Query("date_to"),
		ShiftCode: OptStr(c, "shift"), Types: []string{}, Limit: 10000}
	if d := c.Query("date"); d != "" {
		f.From, f.To = d, d
	}
	if !ValidDate(f.From) || !ValidDate(f.To) || f.To < f.From {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date atau date_from & date_to wajib format YYYY-MM-DD"})
		return f, false
	}
	from, _ := time.Parse("2006-01-02", f.From)
	to, _ := time.Parse("2006-01-02", f.To)
	if to.Sub(from) > 92*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rentang maksimal 93 hari"})
		return f, false
	}
	if v := c.Query("type"); v != "" {
		f.Types = strings.Split(v, ",")
	}
	if v := c.Query("equipment_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "equipment_id bukan UUID"})
			return f, false
		}
		f.EquipmentID = &id
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v < f.Limit {
		f.Limit = v
	}
	return f, true
}
