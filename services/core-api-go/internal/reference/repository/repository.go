// Package repository (fitur reference) = SQL ke tabel core.reference_values (daftar pilihan).
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"simha/core-api/internal/reference/domain"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// List = pilihan aktif satu kelompok (kosong = semua kelompok).
func (r *Repository) List(ctx context.Context, group string) ([]domain.ReferenceValue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_code, code, label, sort_order FROM core.reference_values
		 WHERE is_active AND ($1::text = '' OR group_code = $1)
		 ORDER BY group_code, sort_order, label`, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ReferenceValue{}
	for rows.Next() {
		var v domain.ReferenceValue
		if err := rows.Scan(&v.GroupCode, &v.Code, &v.Label, &v.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
