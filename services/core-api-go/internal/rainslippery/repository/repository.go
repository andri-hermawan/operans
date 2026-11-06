// Package repository (fitur rainslippery) = semua SQL ke tabel core.rain_slippery_logs
// (kode & nama shift dibaca lewat JOIN ke core.shifts).
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/rainslippery/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/dailyinput"
	"simha/core-api/internal/shared/database"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Kunci "data sama ditimpa" (ADR 0007/0015): project + tanggal + shift + lokasi.
var table = database.Table{
	Name: "core.rain_slippery_logs",
	Key:  "uq_rain_slippery_logs_key",
	Cols: []string{"operational_date", "shift_id", "location",
		"rain_start_at", "rain_stop_at", "rain_hours",
		"wait_slippery_start_at", "wait_slippery_stop_at", "wait_slippery_hours",
		"slippery_start_at", "slippery_stop_at", "slippery_hours", "rainfall_mm"},
	Casts: map[string]string{"operational_date": "::text::date"},
}

var (
	upsertSQL = table.UpsertSQL()
	updateSQL = table.UpdateSQL()
)

const selectSQL = `
SELECT t.id, t.project_id, t.operational_date::text, t.shift_id, s.shift_code, s.shift_name, t.location,
       t.rain_start_at, t.rain_stop_at, t.rain_hours::float8,
       t.wait_slippery_start_at, t.wait_slippery_stop_at, t.wait_slippery_hours::float8,
       t.slippery_start_at, t.slippery_stop_at, t.slippery_hours::float8, t.rainfall_mm::float8, t.updated_at
  FROM core.rain_slippery_logs t
  JOIN core.shifts s ON s.id = t.shift_id`

func scan(row pgx.Row) (*domain.RainSlipperyLog, error) {
	var t domain.RainSlipperyLog
	err := row.Scan(&t.ID, &t.ProjectID, &t.OperationalDate, &t.ShiftID, &t.ShiftCode, &t.ShiftName, &t.Location,
		&t.RainStartAt, &t.RainStopAt, &t.RainHours,
		&t.WaitSlipperyStartAt, &t.WaitSlipperyStopAt, &t.WaitSlipperyHours,
		&t.SlipperyStartAt, &t.SlipperyStopAt, &t.SlipperyHours, &t.RainfallMm, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List: tabel ini tidak punya unit, jadi filter unit diabaikan.
func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.RainSlipperyLog, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR t.project_id = $1)
		   AND ($2::text = '' OR t.operational_date >= $2::text::date)
		   AND ($3::text = '' OR t.operational_date <= $3::text::date)
		   AND ($4::uuid IS NULL OR t.shift_id = $4)
		 ORDER BY t.operational_date DESC, s.sequence, t.location
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo, f.ShiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RainSlipperyLog{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.RainSlipperyLog, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE t.id = $1`, id))
}

// args = nilai sesuai urutan table.Cols, diawali project_id dan diakhiri pengguna.
func args(t domain.RainSlipperyLog, by uuid.UUID) []any {
	return []any{t.ProjectID, t.OperationalDate, t.ShiftID, t.Location,
		t.RainStartAt, t.RainStopAt, t.RainHours,
		t.WaitSlipperyStartAt, t.WaitSlipperyStopAt, t.WaitSlipperyHours,
		t.SlipperyStartAt, t.SlipperyStopAt, t.SlipperyHours, t.RainfallMm, by}
}

func (r *Repository) Upsert(ctx context.Context, t domain.RainSlipperyLog, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(t, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, t domain.RainSlipperyLog, by uuid.UUID) error {
	_, err := r.pool.Exec(ctx, updateSQL, append([]any{t.ID}, args(t, by)[1:]...)...) // project_id tidak diubah
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: data hujan & licin lokasi itu pada tanggal dan shift tersebut sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.rain_slippery_logs WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.RainSlipperyLog, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, t := range rows {
		a[i] = args(t, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
