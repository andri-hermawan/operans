// Package repository (fitur monthlyplan) = semua SQL ke tabel core.monthly_plans.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/monthlyplan/domain"
	"simha/core-api/internal/shared/apperror"
	"simha/core-api/internal/shared/database"
	"simha/core-api/internal/shared/plan"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const fixedCols = 8 // project_id, partner, plan_year, plan_month, description, plan_name, uom, total

var (
	dayList   = strings.Join(domain.DayColumns, ", ")
	selectSQL = `SELECT id, project_id, partner, plan_year, plan_month, description, plan_name, uom, total::float8, ` + float8List() + `, updated_at
  FROM core.monthly_plans`
)

func float8List() string {
	cols := make([]string, len(domain.DayColumns))
	for i, c := range domain.DayColumns {
		cols[i] = c + "::float8"
	}
	return strings.Join(cols, ", ")
}

func scan(row pgx.Row) (*domain.MonthlyPlan, error) {
	var p domain.MonthlyPlan
	dest := []any{&p.ID, &p.ProjectID, &p.Partner, &p.PlanYear, &p.PlanMonth, &p.Description, &p.PlanName, &p.Uom, &p.Total}
	for i := range p.Days {
		dest = append(dest, &p.Days[i])
	}
	err := row.Scan(append(dest, &p.UpdatedAt)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) List(ctx context.Context, f plan.Filter) ([]domain.MonthlyPlan, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR project_id = $1)
		   AND ($2::int = 0 OR plan_year = $2)
		   AND ($3::int = 0 OR plan_month = $3)
		   AND ($4::text = '' OR partner = $4)
		 ORDER BY plan_year DESC, plan_month DESC, partner, description, plan_name
		 LIMIT 5000`, f.ProjectID, f.Year, f.Month, f.Partner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.MonthlyPlan{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.MonthlyPlan, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE id = $1`, id))
}

// Kunci "data sama ditimpa" (ADR 0007): project + mitra + tahun + bulan + deskripsi + nama.
var upsertSQL = func() string {
	n := len(domain.DayColumns)
	params, sets := make([]string, n), make([]string, n)
	for i, c := range domain.DayColumns {
		params[i] = fmt.Sprintf("$%d", fixedCols+1+i)
		sets[i] = c + " = EXCLUDED." + c
	}
	by := fixedCols + n + 1
	return `INSERT INTO core.monthly_plans (project_id, partner, plan_year, plan_month, description, plan_name, uom, total, ` + dayList + `, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, ` + strings.Join(params, ", ") + fmt.Sprintf(`, $%d, $%d)`, by, by) + `
ON CONFLICT ON CONSTRAINT uq_monthly_plans_key
DO UPDATE SET uom = EXCLUDED.uom, total = EXCLUDED.total, ` + strings.Join(sets, ", ") + `, updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`
}()

func args(p domain.MonthlyPlan, by uuid.UUID) []any {
	a := []any{p.ProjectID, p.Partner, p.PlanYear, p.PlanMonth, p.Description, p.PlanName, p.Uom, p.Total}
	for _, v := range p.Days {
		a = append(a, v)
	}
	return append(a, by)
}

func (r *Repository) Upsert(ctx context.Context, p domain.MonthlyPlan, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(p, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, p domain.MonthlyPlan, by uuid.UUID) error {
	sets := make([]string, len(domain.DayColumns))
	for i, c := range domain.DayColumns {
		sets[i] = fmt.Sprintf("%s = $%d", c, fixedCols+1+i) // $1 = id, $2..$8 = isian tetap tanpa project_id, tanggal mulai $9
	}
	a := append([]any{p.ID}, args(p, by)[1:]...) // project_id tidak diubah
	_, err := r.pool.Exec(ctx, `UPDATE core.monthly_plans
		   SET partner = $2, plan_year = $3, plan_month = $4, description = $5, plan_name = $6, uom = $7, total = $8, `+strings.Join(sets, ", ")+fmt.Sprintf(`, updated_by = $%d
		 WHERE id = $1`, fixedCols+len(domain.DayColumns)+1), a...)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: rencana dengan mitra, tahun, bulan, deskripsi, dan nama itu sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.monthly_plans WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.MonthlyPlan, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, p := range rows {
		a[i] = args(p, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
