// Package repository (fitur stockemplacement) = semua SQL ke tabel core.stock_emplacements.
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
	"simha/core-api/internal/stockemplacement/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Kolom angka (ton) dibaca sebagai float8; daftar kolom diambil dari domain.AmountColumns supaya satu sumber.
var (
	amountList = strings.Join(domain.AmountColumns, ", ")
	selectSQL  = `SELECT id, project_id, partner, operational_date::text, info, ` + float8List() + `, updated_at
  FROM core.stock_emplacements`
)

func float8List() string {
	cols := make([]string, len(domain.AmountColumns))
	for i, c := range domain.AmountColumns {
		cols[i] = c + "::float8"
	}
	return strings.Join(cols, ", ")
}

func scan(row pgx.Row) (*domain.StockEmplacement, error) {
	var e domain.StockEmplacement
	dest := []any{&e.ID, &e.ProjectID, &e.Partner, &e.OperationalDate, &e.Info}
	for _, p := range e.Amounts() {
		dest = append(dest, p)
	}
	err := row.Scan(append(dest, &e.UpdatedAt)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repository) List(ctx context.Context, f dailyinput.DailyFilter) ([]domain.StockEmplacement, error) {
	rows, err := r.pool.Query(ctx, selectSQL+`
		 WHERE ($1::uuid IS NULL OR project_id = $1)
		   AND ($2::text = '' OR operational_date >= $2::text::date)
		   AND ($3::text = '' OR operational_date <= $3::text::date)
		 ORDER BY operational_date DESC, partner
		 LIMIT 5000`, f.ProjectID, f.DateFrom, f.DateTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.StockEmplacement{}
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*domain.StockEmplacement, error) {
	return scan(r.pool.QueryRow(ctx, selectSQL+` WHERE id = $1`, id))
}

// Kunci "data sama ditimpa" (ADR 0007): project + partner + tanggal.
var upsertSQL = func() string {
	n := len(domain.AmountColumns)
	params, sets := make([]string, n), make([]string, n)
	for i, c := range domain.AmountColumns {
		params[i] = fmt.Sprintf("$%d", i+5)
		sets[i] = c + " = EXCLUDED." + c
	}
	return `INSERT INTO core.stock_emplacements (project_id, partner, operational_date, info, ` + amountList + `, created_by, updated_by)
VALUES ($1, $2, $3::text::date, $4, ` + strings.Join(params, ", ") + fmt.Sprintf(`, $%d, $%d)`, n+5, n+5) + `
ON CONFLICT ON CONSTRAINT uq_stock_emplacements_key
DO UPDATE SET info = EXCLUDED.info, ` + strings.Join(sets, ", ") + `, updated_by = EXCLUDED.updated_by
RETURNING id, (xmax = 0)`
}()

func args(e domain.StockEmplacement, by uuid.UUID) []any {
	a := []any{e.ProjectID, e.Partner, e.OperationalDate, e.Info}
	for _, p := range e.Amounts() {
		a = append(a, *p)
	}
	return append(a, by)
}

func (r *Repository) Upsert(ctx context.Context, e domain.StockEmplacement, by uuid.UUID) (int64, bool, error) {
	var id int64
	var created bool
	err := r.pool.QueryRow(ctx, upsertSQL, args(e, by)...).Scan(&id, &created)
	return id, created, err
}

func (r *Repository) Update(ctx context.Context, e domain.StockEmplacement, by uuid.UUID) error {
	sets := make([]string, len(domain.AmountColumns))
	for i, c := range domain.AmountColumns {
		sets[i] = fmt.Sprintf("%s = $%d", c, i+5)
	}
	a := append([]any{e.ID}, args(e, by)[1:]...) // project_id tidak diubah; $1 = id
	_, err := r.pool.Exec(ctx, `UPDATE core.stock_emplacements
		   SET partner = $2, operational_date = $3::text::date, info = $4, `+strings.Join(sets, ", ")+fmt.Sprintf(`, updated_by = $%d
		 WHERE id = $1`, len(domain.AmountColumns)+5), a...)
	if database.IsUniqueViolation(err) {
		return fmt.Errorf("%w: stok partner itu pada tanggal tersebut sudah ada", apperror.ErrDuplicate)
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM core.stock_emplacements WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpsert(ctx context.Context, rows []domain.StockEmplacement, by uuid.UUID, dryRun bool) (int, int, error) {
	a := make([][]any, len(rows))
	for i, e := range rows {
		a[i] = args(e, by)
	}
	return database.BulkUpsert(ctx, r.pool, upsertSQL, a, dryRun)
}
