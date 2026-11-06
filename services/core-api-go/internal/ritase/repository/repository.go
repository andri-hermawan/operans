// Package repository (fitur ritase) = semua SQL ke tabel core.ritases
// (kode & nama shift dibaca lewat JOIN ke core.shifts).
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/ritase/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Kunci "data sama ditimpa" (ADR 0007/0015): project + tanggal + shift + jam + loader + hauler + lokasi/material.
var table = database.Table{
	Name: "core.ritases",
	Key:  "uq_ritases_key",
	Cols: []string{"operational_date", "shift_id", "time_slot",
		"loader_code", "loader_equipment_id", "loader_model", "loader_operator_nik", "loader_operator_name", "loader_partner",
		"hauler_code", "hauler_equipment_id", "hauler_model", "hauler_operator_nik", "hauler_operator_name", "hauler_partner",
		"block", "location", "dumping", "material", "sub_material", "seam", "phase",
		"distance", "ritase", "production", "production_adjusted", "distance_volume", "productivity_factor"},
	Casts: map[string]string{"operational_date": "::text::date"},
}

var (
	upsertSQL = table.UpsertSQL()
	updateSQL = table.UpdateSQL()
)

const selectSQL = `
SELECT t.id, t.project_id, t.operational_date::text, t.shift_id, s.shift_code, s.shift_name, t.time_slot,
       t.loader_code, t.loader_equipment_id, t.loader_model, t.loader_operator_nik, t.loader_operator_name, t.loader_partner,
       t.hauler_code, t.hauler_equipment_id, t.hauler_model, t.hauler_operator_nik, t.hauler_operator_name, t.hauler_partner,
       t.block, t.location, t.dumping, t.material, t.sub_material, t.seam, t.phase,
       t.distance::float8, t.ritase::float8, t.production::float8, t.production_adjusted::float8,
       t.distance_volume::float8, t.productivity_factor::float8, t.updated_at
  FROM core.ritases t
  JOIN core.shifts s ON s.id = t.shift_id`

func scan(row pgx.Row) (*domain.Ritase, error) {
	var t domain.Ritase
	err := row.Scan(&t.ID, &t.ProjectID, &t.OperationalDate, &t.ShiftID, &t.ShiftCode, &t.ShiftName, &t.TimeSlot,
		&t.LoaderCode, &t.LoaderEquipmentID, &t.LoaderModel, &t.LoaderOperatorNik, &t.LoaderOperatorName, &t.LoaderPartner,
		&t.HaulerCode, &t.HaulerEquipmentID, &t.HaulerModel, &t.HaulerOperatorNik, &t.HaulerOperatorName, &t.HaulerPartner,
		&t.Block, &t.Location, &t.Dumping, &t.Material, &t.SubMaterial, &t.Seam, &t.Phase,
		&t.Distance, &t.Ritase, &t.Production, &t.ProductionAdjusted, &t.DistanceVolume, &t.ProductivityFactor, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List: filter unit memeriksa loader ATAU hauler.
func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.Ritase, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR t.project_id = $1)
		   AND ($2::text = '' OR t.operational_date >= $2::text::date)
		   AND ($3::text = '' OR t.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR t.shift_id = $4)
		   AND ($5::text = '' OR t.loader_code = $5 OR t.hauler_code = $5)
		 ORDER BY t.operational_date DESC, s.sequence, t.time_slot, t.loader_code, t.hauler_code
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Ritase{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.Ritase, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE t.id = $1`, id))
}

// args = nilai sesuai urutan table.Cols, diawali project_id dan diakhiri pengguna.
func args(t domain.Ritase, by uuid.UUID) []any {
	return []any{t.ProjectID, t.OperationalDate, t.ShiftID, t.TimeSlot,
		t.LoaderCode, t.LoaderEquipmentID, t.LoaderModel, t.LoaderOperatorNik, t.LoaderOperatorName, t.LoaderPartner,
		t.HaulerCode, t.HaulerEquipmentID, t.HaulerModel, t.HaulerOperatorNik, t.HaulerOperatorName, t.HaulerPartner,
		t.Block, t.Location, t.Dumping, t.Material, t.SubMaterial, t.Seam, t.Phase,
		t.Distance, t.Ritase, t.Production, t.ProductionAdjusted, t.DistanceVolume, t.ProductivityFactor, by}
}

func (r *Repository) Upsert(ctx context.Context, t domain.Ritase, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(t, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, t domain.Ritase, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, updateSQL, append([]any{t.ID}, args(t, by)[1:]...)...) // project_id tidak diubah
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: ritase dengan kombinasi tanggal, shift, jam, unit, lokasi & material itu sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.ritases WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.Ritase, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, t := range rows {
		a[i] = args(t, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
