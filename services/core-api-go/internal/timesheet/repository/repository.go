// Package repository (fitur timesheet) = semua SQL ke tabel core.time_sheets
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

	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/timesheet/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Kunci "data sama ditimpa" (ADR 0007): project + tanggal + shift + unit.
var table = database.Table{
	Name: "core.time_sheets",
	Key:  "uq_time_sheets_key",
	Cols: []string{"operational_date", "shift_id", "equipment_code", "equipment_id", "operator_nik", "operator_name",
		"hm_start", "hm_end", "hm_total", "remark"},
	Casts: map[string]string{"operational_date": "::text::date"},
}

var (
	upsertSQL = table.UpsertSQL()
	updateSQL = table.UpdateSQL()
)

const selectSQL = `
SELECT t.id, t.project_id, t.operational_date::text, t.shift_id, s.shift_code, s.shift_name,
       t.equipment_code, t.equipment_id, t.operator_nik, t.operator_name,
       t.hm_start::float8, t.hm_end::float8, t.hm_total::float8, t.remark, t.updated_at
  FROM core.time_sheets t
  JOIN core.shifts s ON s.id = t.shift_id`

func scan(row pgx.Row) (*domain.TimeSheet, error) {
	var t domain.TimeSheet
	err := row.Scan(&t.ID, &t.ProjectID, &t.OperationalDate, &t.ShiftID, &t.ShiftCode, &t.ShiftName,
		&t.EquipmentCode, &t.EquipmentID, &t.OperatorNik, &t.OperatorName, &t.HmStart, &t.HmEnd, &t.HmTotal, &t.Remark, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.TimeSheet, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR t.project_id = $1)
		   AND ($2::text = '' OR t.operational_date >= $2::text::date)
		   AND ($3::text = '' OR t.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR t.shift_id = $4)
		   AND ($5::text = '' OR t.equipment_code = $5)
		 ORDER BY t.operational_date DESC, s.sequence, t.equipment_code
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID, strings.ToUpper(f.EquipmentCode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TimeSheet{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.TimeSheet, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE t.id = $1`, id))
}

// args = nilai sesuai urutan table.Cols, diawali project_id dan diakhiri pengguna.
func args(t domain.TimeSheet, by uuid.UUID) []any {
	return []any{t.ProjectID, t.OperationalDate, t.ShiftID, t.EquipmentCode, t.EquipmentID, t.OperatorNik, t.OperatorName,
		t.HmStart, t.HmEnd, t.HmTotal, t.Remark, by}
}

func (r *Repository) Upsert(ctx context.Context, t domain.TimeSheet, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(t, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, t domain.TimeSheet, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, updateSQL, append([]any{t.ID}, args(t, by)[1:]...)...) // project_id tidak diubah
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: time sheet unit itu pada tanggal dan shift tersebut sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.time_sheets WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.TimeSheet, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, t := range rows {
		a[i] = args(t, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
