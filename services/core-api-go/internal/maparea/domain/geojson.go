// File ini (dulu internal/geo) = membaca file peta GeoJSON (ADR 0003) menjadi daftar zona.
// Hanya memeriksa bentuk & atribut; keabsahan geometri (ST_IsValid) diperiksa PostGIS.
package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const MaxFileBytes = 10 << 20 // 10 MB (peta CHR saat ini ±220 KB)

// Feature = satu area pada file, sudah dipetakan ke kolom geo.zones.
type Feature struct {
	Index      int             // urutan di file, mulai 1 (dipakai di pesan kesalahan)
	SourceFID  *int            // ORIG_FID
	Category   string          // Category
	Name       string          // Segment
	LineColor  *string         // HEX_Line (#RRGGBB huruf besar)
	FillColor  *string         // HEX_Fill
	Priority   int             // Priority (opsional): zona bertumpuk, lebih tinggi menang
	Geometry   json.RawMessage // Polygon / MultiPolygon apa adanya
	Properties map[string]any  // semua atribut asli
}

// Issue = kesalahan/peringatan pada satu feature (Feature 0 = seluruh file).
type Issue struct {
	Feature int    `json:"feature,omitempty"`
	Name    string `json:"name,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type rawFile struct {
	Type     string          `json:"type"`
	CRS      json.RawMessage `json:"crs"`
	Features []rawFeature    `json:"features"`
}

type rawFeature struct {
	Type       string          `json:"type"`
	Properties map[string]any  `json:"properties"`
	Geometry   json.RawMessage `json:"geometry"`
}

type rawGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// Parse membaca FeatureCollection. Kembalian: file asli (JSON ringkas), feature yang lolos, dan kesalahan.
// Kesalahan seluruh file (bukan GeoJSON, CRS bukan WGS84, …) dikembalikan sebagai error.
func Parse(r io.Reader) (raw []byte, features []Feature, issues []Issue, err error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return nil, nil, nil, err
	}
	if len(data) > MaxFileBytes {
		return nil, nil, nil, fmt.Errorf("file maksimal %d MB", MaxFileBytes>>20)
	}
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))
	var f rawFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, nil, errors.New("file bukan GeoJSON yang valid: " + err.Error())
	}
	if f.Type != "FeatureCollection" {
		return nil, nil, nil, errors.New(`file harus GeoJSON "FeatureCollection"`)
	}
	if err := checkCRS(f.CRS); err != nil {
		return nil, nil, nil, err
	}
	if len(f.Features) == 0 {
		return nil, nil, nil, errors.New("file tidak berisi feature (area)")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, nil, nil, err
	}

	for i, rf := range f.Features {
		ft, errs := toFeature(i+1, rf)
		if len(errs) > 0 {
			issues = append(issues, errs...)
			continue
		}
		features = append(features, ft)
	}
	return compact.Bytes(), features, issues, nil
}

// checkCRS: GeoJSON standar = WGS84 (lon, lat). CRS lain (mis. UTM EPSG:32748) ditolak.
func checkCRS(crs json.RawMessage) error {
	if len(crs) == 0 || string(crs) == "null" {
		return nil
	}
	s := strings.ToUpper(string(crs))
	if strings.Contains(s, "CRS84") || strings.Contains(s, "4326") {
		return nil
	}
	return errors.New("koordinat file harus WGS84 (EPSG:4326, derajat lon/lat); ekspor ulang dari GIS dengan CRS WGS84")
}

var hexColor = regexp.MustCompile(`^#?([0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$`)

func toFeature(idx int, rf rawFeature) (Feature, []Issue) {
	ft := Feature{Index: idx, Properties: rf.Properties}
	if ft.Properties == nil {
		ft.Properties = map[string]any{}
	}
	var errs []Issue
	bad := func(field, msg string) { errs = append(errs, Issue{Feature: idx, Field: field, Message: msg}) }

	ft.Category = truncateCheck(prop(ft.Properties, "Category"), "Category", bad)
	ft.Name = truncateCheck(prop(ft.Properties, "Segment", "Name"), "Segment", bad)
	if ft.Category == "" {
		bad("Category", "Category wajib diisi")
	}
	if ft.Name == "" {
		bad("Segment", "Segment (nama area) wajib diisi")
	}
	for _, c := range []struct {
		key string
		dst **string
	}{{"HEX_Line", &ft.LineColor}, {"HEX_Fill", &ft.FillColor}} {
		v := prop(ft.Properties, c.key)
		if v == "" {
			continue
		}
		if !hexColor.MatchString(v) {
			bad(c.key, c.key+" \""+v+"\" bukan warna hex (contoh: #D3FFBE)")
			continue
		}
		col := "#" + strings.ToUpper(strings.TrimPrefix(v, "#"))
		*c.dst = &col
	}
	if v := prop(ft.Properties, "ORIG_FID"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ft.SourceFID = &n
		} else {
			bad("ORIG_FID", "ORIG_FID harus bilangan bulat")
		}
	}
	if v := prop(ft.Properties, "Priority"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= -100 && n <= 100 {
			ft.Priority = n
		} else {
			bad("Priority", "Priority harus bilangan bulat -100..100")
		}
	}

	if rf.Type != "Feature" {
		bad("type", `type harus "Feature"`)
	}
	if msg := checkGeometry(rf.Geometry); msg != "" {
		bad("geometry", msg)
	}
	ft.Geometry = rf.Geometry
	for i := range errs {
		errs[i].Name = ft.Name
	}
	return ft, errs
}

// prop mengambil atribut (nama tidak peka huruf besar) sebagai teks.
func prop(p map[string]any, keys ...string) string {
	for _, key := range keys {
		for k, v := range p {
			if !strings.EqualFold(k, key) || v == nil {
				continue
			}
			switch x := v.(type) {
			case string:
				return strings.TrimSpace(x)
			case float64:
				return strconv.FormatFloat(x, 'f', -1, 64)
			default:
				return strings.TrimSpace(fmt.Sprint(x))
			}
		}
	}
	return ""
}

func truncateCheck(v, field string, bad func(string, string)) string {
	if len([]rune(v)) > 100 {
		bad(field, field+" maks. 100 karakter")
	}
	return v
}

// checkGeometry: hanya Polygon/MultiPolygon, cincin tertutup ≥ 4 titik, koordinat derajat WGS84.
func checkGeometry(g json.RawMessage) string {
	if len(g) == 0 || string(g) == "null" {
		return "geometry kosong"
	}
	var rg rawGeometry
	if err := json.Unmarshal(g, &rg); err != nil {
		return "geometry tidak valid"
	}
	var polygons [][][][]float64
	switch rg.Type {
	case "Polygon":
		var p [][][]float64
		if err := json.Unmarshal(rg.Coordinates, &p); err != nil {
			return "koordinat Polygon tidak valid"
		}
		polygons = [][][][]float64{p}
	case "MultiPolygon":
		if err := json.Unmarshal(rg.Coordinates, &polygons); err != nil {
			return "koordinat MultiPolygon tidak valid"
		}
	default:
		return "geometry harus Polygon atau MultiPolygon (bukan " + rg.Type + ")"
	}
	if len(polygons) == 0 {
		return "geometry tidak berisi area"
	}
	for _, poly := range polygons {
		if len(poly) == 0 {
			return "polygon tanpa cincin luar"
		}
		for _, ring := range poly {
			if len(ring) < 4 {
				return "cincin polygon minimal 4 titik"
			}
			for _, pt := range ring {
				if len(pt) < 2 || math.IsNaN(pt[0]) || math.IsNaN(pt[1]) || pt[0] < -180 || pt[0] > 180 || pt[1] < -90 || pt[1] > 90 {
					return "koordinat di luar derajat WGS84 (lon -180..180, lat -90..90); file mungkin memakai UTM"
				}
			}
			first, last := ring[0], ring[len(ring)-1]
			if first[0] != last[0] || first[1] != last[1] {
				return "cincin polygon tidak tertutup (titik awal ≠ titik akhir)"
			}
		}
	}
	return ""
}
