// Package repository = SQL fitur notification: antrian kiriman WhatsApp (tabel iot.notifications).
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simha/iot-ingestion/internal/notification/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// EnqueueNotifications memasukkan kiriman ke antrian (status pending). Pasangan alert + nomor yang sudah ada
// dilewati (tidak dobel). Kembalian = jumlah baris baru.
func (r *Repository) EnqueueNotifications(ctx context.Context, ns []domain.Notification) (int, error) {
	batch := &pgx.Batch{}
	for _, n := range ns {
		batch.Queue(`
			INSERT INTO iot.notifications (alert_id, project_id, recipient_user_id, phone, message)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT ON CONSTRAINT uq_notifications_target DO NOTHING`,
			n.AlertID, n.ProjectID, n.RecipientUserID, n.Phone, n.Message)
	}
	res := r.pool.SendBatch(ctx, batch)
	defer res.Close()
	added := 0
	for range ns {
		tag, err := res.Exec()
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

// ClaimDueNotifications mengambil kiriman pending yang jatuh tempo dan menggeser next_attempt_at sejauh lease
// ("disewa"), sehingga baris yang sama tidak terambil lagi selama sedang dikirim.
func (r *Repository) ClaimDueNotifications(ctx context.Context, limit int, lease time.Duration) ([]domain.Notification, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE iot.notifications n SET next_attempt_at = now() + make_interval(secs => $2), updated_at = now()
		 WHERE n.id IN (SELECT id FROM iot.notifications
		                 WHERE status = 'pending' AND next_attempt_at <= now()
		                 ORDER BY next_attempt_at, id
		                 LIMIT $1 FOR UPDATE SKIP LOCKED)
		RETURNING n.id, n.alert_id, n.project_id, n.recipient_user_id, n.phone, n.message, n.attempts`,
		limit, lease.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.AlertID, &n.ProjectID, &n.RecipientUserID, &n.Phone, &n.Message, &n.Attempts); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) MarkNotificationSent(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE iot.notifications SET status = 'sent', attempts = attempts + 1, sent_at = now(), last_error = NULL,
		       updated_at = now()
		 WHERE id = $1`, id)
	return err
}

// MarkNotificationRetry mencatat percobaan gagal: next = jadwal ulang, nil = gagal permanen (status failed).
func (r *Repository) MarkNotificationRetry(ctx context.Context, id int64, attempts int, errMsg string, next *time.Time) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE iot.notifications
		   SET attempts = $2, last_error = $3, updated_at = now(),
		       status = CASE WHEN $4::timestamptz IS NULL THEN 'failed' ELSE 'pending' END,
		       next_attempt_at = coalesce($4::timestamptz, next_attempt_at)
		 WHERE id = $1`, id, attempts, errMsg, next)
	return err
}
