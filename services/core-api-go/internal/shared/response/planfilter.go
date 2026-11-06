package response

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"simha/core-api/internal/shared/plan"
)

// QueryInt: ?name= angka bulat (kosong → 0). false = sudah dibalas 400.
func QueryInt(c *gin.Context, name string) (int, bool) {
	v := c.Query(name)
	if v == "" {
		return 0, true
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " harus angka"})
		return 0, false
	}
	return n, true
}

// PlanFilter = filter daftar rencana dari query: ?project_id=&year=&month=&partner=. false = sudah dibalas 400.
func PlanFilter(c *gin.Context) (plan.Filter, bool) {
	pid, ok := QueryUUID(c, "project_id")
	if !ok {
		return plan.Filter{}, false
	}
	year, ok := QueryInt(c, "year")
	if !ok {
		return plan.Filter{}, false
	}
	month, ok := QueryInt(c, "month")
	if !ok {
		return plan.Filter{}, false
	}
	return plan.Filter{ProjectID: pid, Year: year, Month: month, Partner: c.Query("partner")}, true
}
