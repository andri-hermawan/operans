// Package service (fitur maparea) = peta project (Map Area, ADR 0003): upload GeoJSON berversi,
// peta untuk frontend, cari zona sebuah titik. Upload hanya admin project (& superadmin); lihat semua role.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"simha/core-api/internal/maparea/domain"
	projectdomain "simha/core-api/internal/project/domain"
	"simha/core-api/internal/shared/access"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/validate"
)

// Repository = kemampuan database yang dibutuhkan (dipenuhi *repository.Repository).
type Repository interface {
	CheckGeometries(ctx context.Context, geoms []json.RawMessage) ([]domain.GeometryCheck, error)
	ReplaceZones(ctx context.Context, projectID uuid.UUID, fileName string, raw []byte, note *string,
		features []domain.Feature, by uuid.UUID, dryRun bool) (uuid.UUID, int, error)
	ZonesGeoJSON(ctx context.Context, projectID uuid.UUID, category string, at *time.Time) ([]byte, error)
	FindZone(ctx context.Context, projectID uuid.UUID, lon, lat float64, at *time.Time) (*domain.Zone, error)
	ListLayerImports(ctx context.Context, projectID uuid.UUID) ([]domain.LayerImport, error)
	LayerImportFile(ctx context.Context, id uuid.UUID) (uuid.UUID, string, []byte, error)
	ZoneExists(ctx context.Context, projectID uuid.UUID, name string) (bool, bool, error)
}

// ProjectFinder = fitur project.
type ProjectFinder interface {
	FindActive(ctx context.Context, id uuid.UUID) (*projectdomain.Project, error)
}

type Service struct {
	repo     Repository
	projects ProjectFinder
}

func New(repo Repository, projects ProjectFinder) *Service {
	return &Service{repo: repo, projects: projects}
}

// ZoneExists: apakah nama zona ada di peta project yang berlaku. hasMap=false bila project belum punya peta.
// Dipakai fitur alertrule untuk memeriksa nama zona.
func (s *Service) ZoneExists(ctx context.Context, projectID uuid.UUID, name string) (exists, hasMap bool, err error) {
	return s.repo.ZoneExists(ctx, projectID, name)
}

// ImportResult = hasil pratinjau / simpan upload peta.
type ImportResult struct {
	DryRun          bool            `json:"dry_run"`
	Saved           bool            `json:"saved"`
	ImportID        *uuid.UUID      `json:"import_id"` // terisi bila disimpan
	FileName        string          `json:"file_name"`
	FeatureCount    int             `json:"feature_count"`
	ValidFeatures   int             `json:"valid_features"`
	InvalidFeatures int             `json:"invalid_features"`
	FixedGeometries int             `json:"fixed_geometries"` // tidak valid, diperbaiki otomatis (ST_MakeValid)
	ReplacedZones   int             `json:"replaced_zones"`   // zona peta lama yang ditutup (valid_to diisi)
	Categories      []CategoryCount `json:"categories"`
	Errors          []domain.Issue  `json:"errors"`
	Warnings        []domain.Issue  `json:"warnings"`
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// Import memeriksa file peta, lalu (bila dryRun=false dan tidak ada kesalahan) mengganti peta project:
// zona lama ditutup, zona baru berlaku mulai sekarang. Hanya admin project (& superadmin).
func (s *Service) Import(ctx context.Context, a access.Actor, projectID *uuid.UUID, filename string, note *string,
	file io.Reader, dryRun bool) (*ImportResult, error) {

	pid, err := s.manageable(ctx, a, projectID)
	if err != nil {
		return nil, err
	}
	if ext := strings.ToLower(filepath.Ext(filename)); ext != ".geojson" && ext != ".json" {
		return nil, apperror.Invalid("file peta harus .geojson atau .json (GeoJSON, WGS84)")
	}
	if note = validate.TrimPtr(note); note != nil && *note == "" {
		note = nil
	}
	if note != nil && len([]rune(*note)) > 255 {
		return nil, apperror.Invalid("note maks. 255 karakter")
	}
	raw, feats, issues, err := domain.Parse(file)
	if err != nil {
		return nil, apperror.Invalid("%s", err.Error())
	}

	res := &ImportResult{DryRun: dryRun, FileName: filename, Errors: []domain.Issue{}, Warnings: []domain.Issue{},
		Categories: []CategoryCount{}}
	res.Errors = append(res.Errors, issues...)
	res.InvalidFeatures = countFeatures(issues)
	res.FeatureCount = len(feats) + res.InvalidFeatures

	geoms := make([]json.RawMessage, len(feats))
	for i, f := range feats {
		geoms[i] = f.Geometry
	}
	checks, err := s.repo.CheckGeometries(ctx, geoms)
	if errors.Is(err, domain.ErrInvalidGeometry) {
		return nil, apperror.Invalid("%s", err.Error())
	}
	if err != nil {
		return nil, err
	}
	var ok []domain.Feature
	seen := map[string]int{}
	byCategory := map[string]int{}
	for i, f := range feats {
		c := checks[i]
		switch {
		case c.Empty:
			res.InvalidFeatures++
			res.Errors = append(res.Errors, domain.Issue{Feature: f.Index, Name: f.Name, Field: "geometry",
				Message: "geometri rusak dan tidak bisa diperbaiki: " + c.Reason})
			continue
		case !c.Valid:
			res.FixedGeometries++
			res.Warnings = append(res.Warnings, domain.Issue{Feature: f.Index, Name: f.Name, Field: "geometry",
				Message: "geometri diperbaiki otomatis (" + c.Reason + ")"})
		}
		key := strings.ToLower(f.Category + "|" + f.Name)
		if prev, dup := seen[key]; dup {
			res.Warnings = append(res.Warnings, domain.Issue{Feature: f.Index, Name: f.Name,
				Message: fmt.Sprintf("kategori & nama sama dengan feature %d (tetap disimpan sebagai area terpisah)", prev)})
		} else {
			seen[key] = f.Index
		}
		byCategory[f.Category]++
		ok = append(ok, f)
	}
	res.ValidFeatures = len(ok)
	for c, n := range byCategory {
		res.Categories = append(res.Categories, CategoryCount{Category: c, Count: n})
	}
	sort.Slice(res.Categories, func(i, j int) bool { return res.Categories[i].Category < res.Categories[j].Category })

	if res.InvalidFeatures > 0 {
		if dryRun {
			return res, nil
		}
		return res, apperror.ErrImportHasErrors
	}
	id, replaced, err := s.repo.ReplaceZones(ctx, pid, filename, raw, note, ok, a.UserID, dryRun)
	if err != nil {
		return nil, err
	}
	res.ReplacedZones = replaced
	if !dryRun {
		res.Saved, res.ImportID = true, &id
	}
	return res, nil
}

// countFeatures = jumlah feature berbeda yang punya kesalahan.
func countFeatures(issues []domain.Issue) int {
	set := map[int]bool{}
	for _, i := range issues {
		set[i.Feature] = true
	}
	return len(set)
}

// Zones = peta yang berlaku (atau pada waktu at) sebagai GeoJSON FeatureCollection. Semua role project.
func (s *Service) Zones(ctx context.Context, a access.Actor, projectID *uuid.UUID, category string, at *time.Time) ([]byte, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.ZonesGeoJSON(ctx, pid, strings.TrimSpace(category), at)
}

// Lookup = zona tempat sebuah titik (lat, lon) berada; nil bila di luar semua zona.
func (s *Service) Lookup(ctx context.Context, a access.Actor, projectID *uuid.UUID, lat, lon float64, at *time.Time) (*domain.Zone, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, apperror.Invalid("lat harus -90..90 dan lon -180..180")
	}
	return s.repo.FindZone(ctx, pid, lon, lat, at)
}

// Imports = riwayat upload peta. Semua role project boleh melihat.
func (s *Service) Imports(ctx context.Context, a access.Actor, projectID *uuid.UUID) ([]domain.LayerImport, error) {
	pid, err := access.Viewable(a, projectID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListLayerImports(ctx, pid)
}

// ImportFile = file GeoJSON asli sebuah upload (untuk diunduh / diunggah ulang sebagai rollback).
func (s *Service) ImportFile(ctx context.Context, a access.Actor, id uuid.UUID) (string, []byte, error) {
	pid, name, raw, err := s.repo.LayerImportFile(ctx, id)
	if err != nil {
		return "", nil, err
	}
	if raw == nil || !a.CanView(pid) {
		return "", nil, apperror.ErrNotFound
	}
	return name, raw, nil
}

func (s *Service) manageable(ctx context.Context, a access.Actor, requested *uuid.UUID) (uuid.UUID, error) {
	pid, err := access.TargetProject(a, requested)
	if err != nil {
		return uuid.Nil, err
	}
	if !a.CanManageProject(pid) {
		return uuid.Nil, apperror.ErrForbidden
	}
	p, err := s.projects.FindActive(ctx, pid)
	if err != nil {
		return uuid.Nil, err
	}
	if p == nil {
		return uuid.Nil, apperror.Invalid("project tidak ditemukan atau nonaktif")
	}
	return pid, nil
}
