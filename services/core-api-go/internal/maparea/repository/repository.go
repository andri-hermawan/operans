// Package repository (fitur maparea) = semua SQL ke tabel geo.layer_imports & geo.zones (PostGIS).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/maparea/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// rawGeomSQL = geometri GeoJSON ($1) apa adanya, WGS84.
const rawGeomSQL = `ST_SetSRID(ST_GeomFromGeoJSON($1::text), 4326)`

// fixGeom: → MultiPolygon 2D yang valid (diperbaiki ST_MakeValid bila perlu, hanya bagian area yang diambil).
func fixGeom(expr string) string {
	return `ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_Force2D(` + expr + `)), 3))`
}

// CheckGeometries memeriksa semua geometri sekaligus (satu batch) tanpa menyimpan apa pun.
func (r *Repository) CheckGeometries(ctx context.Context, geoms []json.RawMessage) ([]domain.GeometryCheck, error) {
	batch := &pgx.Batch{}
	for _, g := range geoms {
		batch.Queue(`
			WITH g AS (SELECT `+rawGeomSQL+` AS raw)
			SELECT ST_IsValid(raw), coalesce(nullif(ST_IsValidReason(raw), 'Valid Geometry'), ''), ST_IsEmpty(`+fixGeom("raw")+`)
			  FROM g`, string(g))
	}
	res := r.pool.SendBatch(ctx, batch)
	defer res.Close()
	out := make([]domain.GeometryCheck, len(geoms))
	for i := range geoms {
		if err := res.QueryRow().Scan(&out[i].Valid, &out[i].Reason, &out[i].Empty); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) { // PostGIS menolak bentuk geometri
				return nil, fmt.Errorf("%w (feature %d): %s", domain.ErrInvalidGeometry, i+1, pgErr.Message)
			}
			return nil, err
		}
	}
	return out, nil
}

// ReplaceZones menyimpan upload peta dalam SATU transaksi: arsip file, menutup zona yang sedang
// berlaku (valid_to = sekarang), lalu memasukkan zona baru (valid_from = sekarang).
// dryRun=true → dijalankan lalu DIBATALKAN. Kembalian: id upload & jumlah zona lama yang ditutup.
func (r *Repository) ReplaceZones(ctx context.Context, projectID uuid.UUID, fileName string, raw []byte, note *string,
	features []domain.Feature, by uuid.UUID, dryRun bool) (importID uuid.UUID, replaced int, err error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer tx.Rollback(ctx)

	if err = tx.QueryRow(ctx, `
		INSERT INTO geo.layer_imports (project_id, file_name, raw_geojson, feature_count, note, imported_by)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6) RETURNING id`,
		projectID, fileName, string(raw), len(features), note, by).Scan(&importID); err != nil {
		return uuid.Nil, 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE geo.zones SET valid_to = now() WHERE project_id = $1 AND valid_to IS NULL`, projectID)
	if err != nil {
		return uuid.Nil, 0, err
	}
	replaced = int(tag.RowsAffected())

	batch := &pgx.Batch{}
	for _, f := range features {
		props, err := json.Marshal(f.Properties)
		if err != nil {
			return uuid.Nil, 0, err
		}
		batch.Queue(`
			WITH g AS (SELECT `+fixGeom(rawGeomSQL)+` AS geom)
			INSERT INTO geo.zones (project_id, import_id, source_fid, category, name, line_color, fill_color,
			                       priority, geom, area_m2, properties, valid_from, created_by)
			SELECT $2, $3, $4, $5, $6, $7, $8, $9, g.geom, round(ST_Area(g.geom::geography)::numeric, 2),
			       $10::jsonb, now(), $11
			  FROM g`,
			string(f.Geometry), projectID, importID, f.SourceFID, f.Category, f.Name, f.LineColor, f.FillColor,
			f.Priority, string(props), by)
	}
	if err = tx.SendBatch(ctx, batch).Close(); err != nil {
		return uuid.Nil, 0, err
	}
	if dryRun {
		return importID, replaced, nil
	}
	return importID, replaced, tx.Commit(ctx)
}

// ZonesGeoJSON = zona yang berlaku pada waktu at, sebagai GeoJSON FeatureCollection (dibangun PostGIS).
// at nil = sekarang. category kosong = semua kategori.
func (r *Repository) ZonesGeoJSON(ctx context.Context, projectID uuid.UUID, category string, at *time.Time) ([]byte, error) {
	var out []byte
	err := r.pool.QueryRow(ctx, `
		SELECT json_build_object('type', 'FeatureCollection', 'features', coalesce(json_agg(json_build_object(
		         'type', 'Feature', 'id', z.id,
		         'geometry', ST_AsGeoJSON(z.geom, 7)::json,
		         'properties', json_build_object(
		             'id', z.id, 'category', z.category, 'name', z.name, 'line_color', z.line_color,
		             'fill_color', z.fill_color, 'priority', z.priority, 'source_fid', z.source_fid,
		             'area_m2', z.area_m2, 'attributes', z.properties))
		         ORDER BY z.category, z.name), '[]'::json))::text
		  FROM geo.zones z
		 WHERE z.project_id = $1
		   AND ($2::text = '' OR z.category = $2)
		   AND z.valid_from <= coalesce($3::timestamptz, now())
		   AND (z.valid_to IS NULL OR z.valid_to > coalesce($3::timestamptz, now()))`,
		projectID, category, at).Scan(&out)
	return out, err
}

// FindZone = zona tempat titik berada (aturan tumpuk ada di fungsi geo.find_zone). nil bila di luar semua zona.
func (r *Repository) FindZone(ctx context.Context, projectID uuid.UUID, lon, lat float64, at *time.Time) (*domain.Zone, error) {
	var z domain.Zone
	err := r.pool.QueryRow(ctx, `
		SELECT id, project_id, import_id, source_fid, category, name, line_color, fill_color, priority,
		       area_m2::float8, valid_from, valid_to
		  FROM geo.find_zone($1, $2, $3, coalesce($4::timestamptz, now()))`, projectID, lon, lat, at).
		Scan(&z.ID, &z.ProjectID, &z.ImportID, &z.SourceFID, &z.Category, &z.Name, &z.LineColor, &z.FillColor,
			&z.Priority, &z.AreaM2, &z.ValidFrom, &z.ValidTo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &z, nil
}

// ListLayerImports = riwayat upload peta project (terbaru dulu).
func (r *Repository) ListLayerImports(ctx context.Context, projectID uuid.UUID) ([]domain.LayerImport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.project_id, i.file_name, i.feature_count, i.note, i.imported_by, u.name, i.imported_at,
		       EXISTS (SELECT 1 FROM geo.zones z WHERE z.import_id = i.id AND z.valid_to IS NULL)
		  FROM geo.layer_imports i
		  LEFT JOIN auth.users u ON u.id = i.imported_by
		 WHERE i.project_id = $1
		 ORDER BY i.imported_at DESC
		 LIMIT 200`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.LayerImport{}
	for rows.Next() {
		var l domain.LayerImport
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.FileName, &l.FeatureCount, &l.Note, &l.ImportedBy,
			&l.ImportedByName, &l.ImportedAt, &l.IsCurrent); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// LayerImportFile = file GeoJSON asli sebuah upload. raw nil bila tidak ada.
func (r *Repository) LayerImportFile(ctx context.Context, id uuid.UUID) (projectID uuid.UUID, fileName string, raw []byte, err error) {
	err = r.pool.QueryRow(ctx, `SELECT project_id, file_name, raw_geojson::text FROM geo.layer_imports WHERE id = $1`, id).
		Scan(&projectID, &fileName, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", nil, nil
	}
	return projectID, fileName, raw, err
}

// ZoneExists: apakah nama zona ada di peta project yang berlaku. hasMap=false bila project belum punya peta.
func (r *Repository) ZoneExists(ctx context.Context, projectID uuid.UUID, name string) (exists, hasMap bool, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT coalesce(bool_or(lower(name) = lower($2)), false), count(*) > 0
		  FROM geo.zones WHERE project_id = $1 AND valid_to IS NULL`, projectID, name).Scan(&exists, &hasMap)
	return exists, hasMap, err
}
