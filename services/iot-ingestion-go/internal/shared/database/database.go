// Package database membuat koneksi ke PostgreSQL (pgx pool) dan berisi pembantu kecil yang dipakai
// repository semua fitur (cek galat koneksi, cek "data sudah ada").
// Repository tiap fitur menerima *pgxpool.Pool dari sini lalu menjalankan SQL-nya sendiri.
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect membuat pool sekali saat program start, lalu memastikan database benar-benar terjangkau.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10                      // di bawah CONNECTION LIMIT 20 milik iot_ingest
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

// IsConnectionError: galat jaringan/koneksi (bukan galat SQL dari server, yang tidak akan berubah bila diulang).
func IsConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	return !errors.As(err, &pgErr)
}

// IsUniqueViolation: galat "data sudah ada" (kode PostgreSQL 23505, melanggar index unik).
func IsUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
