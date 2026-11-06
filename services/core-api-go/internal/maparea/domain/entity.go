// Package domain (fitur maparea) = peta project: riwayat upload GeoJSON, zona (area) beserta
// masa berlakunya, dan pembaca file GeoJSON (geojson.go).
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidGeometry = PostGIS menolak bentuk geometri (dulu di domain/users.go).
var ErrInvalidGeometry = errors.New("geometri tidak bisa dibaca") // dibungkus pesan PostGIS
// LayerImport = satu kali upload peta GeoJSON (tabel geo.layer_imports).
type LayerImport struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	FileName       string     `json:"file_name"`
	FeatureCount   int        `json:"feature_count"`
	Note           *string    `json:"note"`
	ImportedBy     *uuid.UUID `json:"imported_by"`
	ImportedByName *string    `json:"imported_by_name"`
	ImportedAt     time.Time  `json:"imported_at"`
	IsCurrent      bool       `json:"is_current"` // zona dari upload ini yang sedang berlaku
}

// Zone = satu area peta (tabel geo.zones), tanpa geometri.
type Zone struct {
	ID        uuid.UUID  `json:"id"`
	ProjectID uuid.UUID  `json:"project_id"`
	ImportID  uuid.UUID  `json:"import_id"`
	SourceFID *int       `json:"source_fid"`
	Category  string     `json:"category"`
	Name      string     `json:"name"`
	LineColor *string    `json:"line_color"`
	FillColor *string    `json:"fill_color"`
	Priority  int        `json:"priority"`
	AreaM2    float64    `json:"area_m2"`
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
}

// GeometryCheck = hasil pemeriksaan PostGIS untuk satu geometri feature.
type GeometryCheck struct {
	Valid  bool   // ST_IsValid apa adanya
	Reason string // ST_IsValidReason bila tidak valid
	Empty  bool   // setelah diperbaiki (ST_MakeValid) tidak tersisa area → ditolak
}
