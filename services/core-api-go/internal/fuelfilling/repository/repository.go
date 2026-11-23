// Package repository (fitur fuelfilling) = semua SQL ke tabel core.fuel_fillings
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

	"simha/core-api/internal/fuelfilling/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Kunci "data sama ditimpa": project + tanggal + shift + unit + HM pengisian.
var table = database.Table{
	Name: "core.fuel_fillings",
	Key:  "uq_fuel_fillings_key",
	Cols: []string{"operational_date", "shift_id", "equipment_code", "equipment_id", "equipment_model", "hm_previous", "hm_filling",
		"total_hours", "fuel_usage_liter", "fuel_consumption_lph", "waste_removal_liter", "coal_mining_liter", "hauling_liter",
		"rental_liter", "dewatering_liter", "final_support_liter"},
	Casts: map[string]string{"operational_date": "::text::date"},
}

var (
	upsertSQL = table.UpsertSQL()
	updateSQL = table.UpdateSQL()
)

const selectSQL = `
SELECT f.id, f.project_id, f.operational_date::text, f.shift_id, s.shift_code, s.shift_name,
       f.equipment_code, f.equipment_id, f.equipment_model,
       f.hm_previous::float8, f.hm_filling::float8, f.total_hours::float8, f.fuel_usage_liter::float8, f.fuel_consumption_lph::float8,
       f.waste_removal_liter::float8, f.coal_mining_liter::float8, f.hauling_liter::float8, f.rental_liter::float8,
       f.dewatering_liter::float8, f.final_support_liter::float8, f.updated_at
  FROM core.fuel_fillings f
  JOIN core.shifts s ON s.id = f.shift_id`

func scan(row pgx.Row) (*domain.FuelFilling, error) {
	var f domain.FuelFilling
	err := row.Scan(&f.ID, &f.ProjectID, &f.OperationalDate, &f.ShiftID, &f.ShiftCode, &f.ShiftName,
		&f.EquipmentCode, &f.EquipmentID, &f.EquipmentModel, &f.HmPrevious, &f.HmFilling, &f.TotalHours, &f.FuelUsage, &f.FuelConsumption,
		&f.WasteRemoval, &f.CoalMining, &f.Hauling, &f.Rental, &f.Dewatering, &f.FinalSupport, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.FuelFilling, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR f.project_id = $1)
		   AND ($2::text = '' OR f.operational_date >= $2::text::date)
		   AND ($3::text = '' OR f.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR f.shift_id = $4)
		   AND ($5::text = '' OR f.equipment_code = $5)
		 ORDER BY f.operational_date DESC, s.sequence, f.equipment_code, f.hm_filling NULLS LAST
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FuelFilling{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *x)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.FuelFilling, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE f.id = $1`, id))
}

// args = nilai sesuai urutan table.Cols, diawali project_id dan diakhiri pengguna.
func args(f domain.FuelFilling, by uuid.UUID) []any {
	return []any{f.ProjectID, f.OperationalDate, f.ShiftID, f.EquipmentCode, f.EquipmentID, f.EquipmentModel, f.HmPrevious, f.HmFilling,
		f.TotalHours, f.FuelUsage, f.FuelConsumption, f.WasteRemoval, f.CoalMining, f.Hauling, f.Rental, f.Dewatering, f.FinalSupport, by}
}

func (r *Repository) Upsert(ctx context.Context, f domain.FuelFilling, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(f, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, f domain.FuelFilling, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, updateSQL, append([]any{f.ID}, args(f, by)[1:]...)...) // project_id tidak diubah
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: pengisian unit itu dengan HM yang sama pada tanggal dan shift tersebut sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.fuel_fillings WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.FuelFilling, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, f := range rows {
		a[i] = args(f, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
