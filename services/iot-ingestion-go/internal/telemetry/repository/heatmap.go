package repository

import (
	"context"
	"math"

	"simha/iot-ingestion/internal/shared/database"
	"simha/iot-ingestion/internal/telemetry/domain"
)

// metersPerDegree = panjang 1 derajat lintang (≈ konstan); bujur dikoreksi dengan cos(lintang).
const metersPerDegree = 111320.0

// CellDegrees mengubah sisi kotak (meter) menjadi ukuran derajat di lintang `lat`.
func CellDegrees(cellMeters, lat float64) (dLon, dLat float64) {
	dLat = cellMeters / metersPerDegree
	dLon = cellMeters / (metersPerDegree * math.Cos(lat*math.Pi/180))
	return dLon, dLat
}

// HeatGrid meringkas titik telemetry satu slot (tanggal operasional + shift [+ unit]) menjadi kotak-kotak.
// Query ini TIDAK mengubah/menggantikan Points & SegmentSpeeds; keduanya tetap dipakai halaman lain.
// Kotak dibuat dengan ST_SnapToGrid; hanya titik yang punya lokasi & kecepatan. Bila ada batas peta (bbox), hanya
// titik di dalamnya. Kotak terpadat diutamakan bila jumlahnya melebihi `Limit`.
func (r *Repository) HeatGrid(ctx context.Context, s database.Slot, q domain.HeatGridQuery) ([]domain.HeatCell, error) {
	lat0 := 0.0
	if q.HasBBox {
		lat0 = (q.MinLat + q.MaxLat) / 2
	}
	dLon, dLat := CellDegrees(q.CellMeters, lat0)
	sql := `
		SELECT ST_Y(cell), ST_X(cell), n, units, round(avg_speed::numeric, 1)::float8, max_speed,
		       round(avg_empty::numeric, 1)::float8, round(avg_loaded::numeric, 1)::float8
		  FROM (
		    SELECT ST_SnapToGrid(location, $5::float8, $6::float8) AS cell,
		           count(*) AS n, count(DISTINCT equipment_id) AS units,
		           avg(speed_kph) AS avg_speed, max(speed_kph) AS max_speed,
		           avg(speed_kph) FILTER (WHERE vessel_status = 'empty')  AS avg_empty,
		           avg(speed_kph) FILTER (WHERE vessel_status = 'loaded') AS avg_loaded
		      FROM iot.telemetry_events
		     WHERE equipment_id IS NOT NULL AND location IS NOT NULL AND speed_kph IS NOT NULL AND ` + database.SlotWhere + `
		       AND ($7::boolean IS FALSE OR location && ST_MakeEnvelope($8::float8, $9::float8, $10::float8, $11::float8, 4326))
		     GROUP BY 1
		  ) g
		 ORDER BY n DESC LIMIT $12`
	args := append(s.Args(), dLon, dLat, q.HasBBox, q.MinLon, q.MinLat, q.MaxLon, q.MaxLat, q.Limit)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.HeatCell{}
	for rows.Next() {
		var c domain.HeatCell
		if err := rows.Scan(&c.Latitude, &c.Longitude, &c.Points, &c.Units, &c.AvgSpeed, &c.MaxSpeed, &c.AvgEmpty, &c.AvgLoaded); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
