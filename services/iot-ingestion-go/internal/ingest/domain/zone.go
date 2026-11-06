package domain

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// Zone = satu area peta project (dari geo.v_iot_zones, ADR 0003), geometri disimpan di memori.
type Zone struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	SourceFID *int32 // ORIG_FID lama (tidak unik); dipakai status muatan
	Category  string
	Name      string
	Priority  int
	AreaM2    float64
	ValidFrom time.Time
	ValidTo   *time.Time // nil = peta yang berlaku sekarang

	polygons                       [][][][2]float64 // polygon → cincin (0 = luar, sisanya lubang) → titik [lon, lat]
	minLon, minLat, maxLon, maxLat float64          // kotak batas: penyaring cepat sebelum hitung poligon
}

// SetGeoJSON mengisi geometri dari teks GeoJSON MultiPolygon (ST_AsGeoJSON) atau Polygon.
func (z *Zone) SetGeoJSON(text string) error {
	var g struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(text), &g); err != nil {
		return err
	}
	var polys [][][][]float64
	switch g.Type {
	case "MultiPolygon":
		if err := json.Unmarshal(g.Coordinates, &polys); err != nil {
			return err
		}
	case "Polygon":
		var p [][][]float64
		if err := json.Unmarshal(g.Coordinates, &p); err != nil {
			return err
		}
		polys = [][][][]float64{p}
	default:
		return errors.New("geometri zona harus Polygon/MultiPolygon")
	}
	z.minLon, z.minLat, z.maxLon, z.maxLat = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	z.polygons = make([][][][2]float64, len(polys))
	for i, poly := range polys {
		z.polygons[i] = make([][][2]float64, len(poly))
		for j, ring := range poly {
			pts := make([][2]float64, 0, len(ring))
			for _, p := range ring {
				if len(p) < 2 {
					return errors.New("titik geometri tidak lengkap")
				}
				pts = append(pts, [2]float64{p[0], p[1]})
				z.minLon, z.maxLon = math.Min(z.minLon, p[0]), math.Max(z.maxLon, p[0])
				z.minLat, z.maxLat = math.Min(z.minLat, p[1]), math.Max(z.maxLat, p[1])
			}
			z.polygons[i][j] = pts
		}
	}
	if len(z.polygons) == 0 {
		return errors.New("geometri zona kosong")
	}
	return nil
}

// Contains: titik (lon, lat) di dalam zona? Garis batas dihitung DI DALAM (sama dengan ST_Intersects /
// turf booleanPointInPolygon di sistem lama).
func (z *Zone) Contains(lon, lat float64) bool {
	if lon < z.minLon || lon > z.maxLon || lat < z.minLat || lat > z.maxLat {
		return false
	}
	for _, poly := range z.polygons {
		if len(poly) == 0 {
			continue
		}
		in, edge := inRing(poly[0], lon, lat)
		if !in && !edge {
			continue
		}
		if edge {
			return true
		}
		inHole := false
		for _, hole := range poly[1:] {
			if h, e := inRing(hole, lon, lat); h && !e {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

// inRing = uji "sinar ke kanan" (jumlah perpotongan ganjil = di dalam); edge = tepat di garis batas.
func inRing(ring [][2]float64, x, y float64) (inside, edge bool) {
	n := len(ring)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi, xj, yj := ring[i][0], ring[i][1], ring[j][0], ring[j][1]
		if onSegment(xi, yi, xj, yj, x, y) {
			return true, true
		}
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside, false
}

func onSegment(x1, y1, x2, y2, x, y float64) bool {
	const eps = 1e-12
	cross := (x2-x1)*(y-y1) - (y2-y1)*(x-x1)
	if math.Abs(cross) > eps {
		return false
	}
	return x >= math.Min(x1, x2)-eps && x <= math.Max(x1, x2)+eps && y >= math.Min(y1, y2)-eps && y <= math.Max(y1, y2)+eps
}

// ValidAt: zona berlaku pada waktu t (versi peta, ADR 0003).
func (z *Zone) ValidAt(t time.Time) bool {
	return !t.Before(z.ValidFrom) && (z.ValidTo == nil || t.Before(*z.ValidTo))
}

// FindZone = zona tempat titik berada pada waktu t. Bertumpuk: priority tertinggi, lalu luas terkecil,
// lalu nama (SAMA dengan fungsi database geo.find_zone). nil = di luar semua zona.
// (Lama: feature TERAKHIR di file yang memuat titik.)
func FindZone(zones []Zone, lon, lat float64, t time.Time) *Zone {
	var best *Zone
	for i := range zones {
		z := &zones[i]
		if !z.ValidAt(t) || !z.Contains(lon, lat) {
			continue
		}
		if best == nil || z.Priority > best.Priority ||
			(z.Priority == best.Priority && (z.AreaM2 < best.AreaM2 || (z.AreaM2 == best.AreaM2 && z.Name < best.Name))) {
			best = z
		}
	}
	return best
}
