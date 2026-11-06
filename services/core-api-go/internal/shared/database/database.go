// Package database = koneksi PostgreSQL (pgx pool) dan bantuan SQL yang dipakai repository semua fitur.
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Open membuat pool sekali saat program start, lalu memastikan database benar-benar terjangkau.
// Pool ini dibagikan ke repository semua fitur (lihat cmd/api/main.go). Tutup dengan pool.Close().
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10                      // di bawah CONNECTION LIMIT 20 milik core_api
	cfg.MaxConnLifetime = 30 * time.Minute // sambungan diganti berkala agar tidak "basi"
	cfg.MaxConnIdleTime = 5 * time.Minute  // sambungan nganggur ditutup

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// IsUniqueViolation: galat PostgreSQL "data sudah ada" (kode 23505, melanggar UNIQUE).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// BulkUpsert menjalankan satu SQL upsert untuk banyak baris (pgx Batch) dalam satu transaksi.
// SQL wajib RETURNING id, (xmax = 0) agar baris baru vs diperbarui bisa dihitung.
// dryRun=true → dijalankan lalu DIBATALKAN, hanya untuk menghitung baru vs diperbarui.
// Dipakai impor Excel (operator, aktivasi, breakdown, weighbridge).
func BulkUpsert(ctx context.Context, pool *pgxpool.Pool, sql string, args [][]any, dryRun bool) (created, updated int, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx) // dryRun: selalu dibatalkan; gagal: dibatalkan

	batch := &pgx.Batch{}
	for _, a := range args {
		batch.Queue(sql, a...)
	}
	res := tx.SendBatch(ctx, batch)
	for range args {
		var id int64
		var isNew bool
		if err := res.QueryRow().Scan(&id, &isNew); err != nil {
			res.Close()
			return 0, 0, err
		}
		if isNew {
			created++
		} else {
			updated++
		}
	}
	if err := res.Close(); err != nil {
		return 0, 0, err
	}
	if dryRun {
		return created, updated, nil
	}
	return created, updated, tx.Commit(ctx)
}
