package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"simha/iot-ingestion/internal/shared/response"
	"simha/iot-ingestion/internal/telemetry/domain"
)

const (
	minCellMeters   = 5
	maxCellMeters   = 1000
	defaultCell     = 20
	maxHeatCells    = 30000
	defaultHeatCell = 15000
)

// parseHeatGrid membaca ?cell_m (sisi kotak, meter), ?bbox=minLon,minLat,maxLon,maxLat (opsional), ?limit.
// Mengembalikan pesan galat (kosong bila benar).
func parseHeatGrid(cellM, bbox, limit string) (domain.HeatGridQuery, string) {
	q := domain.HeatGridQuery{CellMeters: defaultCell, Limit: defaultHeatCell}
	if cellM != "" {
		v, err := strconv.ParseFloat(cellM, 64)
		if err != nil || v < minCellMeters || v > maxCellMeters {
			return q, "cell_m harus angka " + strconv.Itoa(minCellMeters) + "–" + strconv.Itoa(maxCellMeters) + " (meter)"
		}
		q.CellMeters = v
	}
	if bbox != "" {
		parts := strings.Split(bbox, ",")
		if len(parts) != 4 {
			return q, "bbox harus minLon,minLat,maxLon,maxLat"
		}
		var n [4]float64
		for i, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return q, "bbox harus angka"
			}
			n[i] = v
		}
		if n[0] >= n[2] || n[1] >= n[3] || n[0] < -180 || n[2] > 180 || n[1] < -90 || n[3] > 90 {
			return q, "bbox tidak valid"
		}
		q.MinLon, q.MinLat, q.MaxLon, q.MaxLat, q.HasBBox = n[0], n[1], n[2], n[3], true
	}
	if limit != "" {
		if v, err := strconv.Atoi(limit); err == nil && v > 0 && v <= maxHeatCells {
			q.Limit = v
		}
	}
	return q, ""
}

// GET /api/v1/telemetry/heat-grid?date&shift&equipment_id&cell_m=&bbox=&limit=
// Satu baris per kotak peta (bukan per titik) → ringan walau unit ratusan. Rata-rata dihitung dari semua titik di kotak.
func (h *Handler) HeatGrid(c *gin.Context) {
	s, ok := response.ReadSlot(c)
	if !ok {
		return
	}
	q, msg := parseHeatGrid(c.Query("cell_m"), c.Query("bbox"), c.Query("limit"))
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	rows, err := h.svc.HeatGrid(c.Request.Context(), s, q)
	if response.InternalErr(c, err, "kotak heatmap gagal") {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows, "count": len(rows), "truncated": len(rows) == q.Limit, "cell_m": q.CellMeters})
}
