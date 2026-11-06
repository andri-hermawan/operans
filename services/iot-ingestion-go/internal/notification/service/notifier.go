// Package service (fitur notification) = notifikasi WhatsApp alert lewat antrian (outbox) iot.notifications (ADR 0009, C5).
//
// Alur: alert DIBUKA → AlertOpened memasukkan satu baris per nomor penerima (status pending) → Worker
// mengambil yang jatuh tempo, mengirim ke server WhatsApp (sama dengan sistem lama: POST {number, message}),
// lalu menandai sent, atau mencoba ulang dengan jeda bertahap sampai batas percobaan (failed).
// Server WA mati / lambat tidak pernah menghambat penyimpanan telemetry & alert.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	ingestdomain "simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/notification/domain"
	"simha/iot-ingestion/internal/shared/masterdata"
)

// Store = antrian di database (dipenuhi *repository.Repository fitur ini).
type Store interface {
	EnqueueNotifications(ctx context.Context, ns []domain.Notification) (int, error)
	// ClaimDueNotifications mengambil kiriman jatuh tempo dan "menyewanya" (next_attempt_at digeser maju)
	// agar tidak terambil dua kali, juga bila kelak ada lebih dari satu instance.
	ClaimDueNotifications(ctx context.Context, limit int, lease time.Duration) ([]domain.Notification, error)
	MarkNotificationSent(ctx context.Context, id int64) error
	MarkNotificationRetry(ctx context.Context, id int64, attempts int, errMsg string, next *time.Time) error // next nil = gagal permanen
}

// Directory = data induk untuk isi pesan (dipenuhi *masterdata.Cache).
type Directory interface {
	Recipients(ids []uuid.UUID) []masterdata.Recipient
	Label(projectID, equipmentID *uuid.UUID) (equipmentCode, projectName string, loc *time.Location)
}

type Config struct {
	URL         string        // WHATSAPP_API_URL; kosong = WhatsApp nonaktif
	Timeout     time.Duration // WHATSAPP_TIMEOUT_MS
	MaxAttempts int           // batas percobaan sebelum "failed"
}

type Notifier struct {
	cfg    Config
	store  Store
	dir    Directory
	client *http.Client
	wake   chan struct{}
}

func New(cfg Config, store Store, dir Directory) *Notifier {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Notifier{cfg: cfg, store: store, dir: dir, client: &http.Client{Timeout: cfg.Timeout}, wake: make(chan struct{}, 1)}
}

func (n *Notifier) Enabled() bool { return n.cfg.URL != "" }

// AlertOpened memasukkan pesan alert ke antrian untuk setiap nomor penerima (satu nomor satu pesan).
// Tidak pernah menggagalkan pemanggil: galat hanya dicatat di log.
func (n *Notifier) AlertOpened(ctx context.Context, a ingestdomain.Alert) {
	if !n.Enabled() || len(a.Recipients) == 0 {
		return
	}
	code, project, loc := n.dir.Label(a.ProjectID, &a.EquipmentID)
	msg := domain.AlertMessage(a, code, project, loc)
	seen := map[string]bool{}
	var ns []domain.Notification
	for _, r := range n.dir.Recipients(a.Recipients) {
		phone := domain.NormalizePhone(r.Phone)
		if phone == "" || seen[phone] {
			continue
		}
		seen[phone] = true
		uid := r.UserID
		ns = append(ns, domain.Notification{AlertID: a.ID, ProjectID: a.ProjectID, RecipientUserID: &uid, Phone: phone, Message: msg})
	}
	if len(ns) == 0 {
		return
	}
	if _, err := n.store.EnqueueNotifications(ctx, ns); err != nil {
		slog.Error("antrian notifikasi gagal", "alert_id", a.ID, "err", err)
		return
	}
	select { // bangunkan pengirim agar pesan keluar segera
	case n.wake <- struct{}{}:
	default:
	}
}

// Run = pengirim: memeriksa antrian tiap `every` (atau segera saat ada pesan baru) sampai ctx berakhir.
func (n *Notifier) Run(ctx context.Context, every time.Duration) {
	if !n.Enabled() {
		slog.Info("WhatsApp nonaktif (WHATSAPP_API_URL kosong): alert tetap tersimpan & tampil live, tanpa pesan WA")
		return
	}
	slog.Info("pengirim WhatsApp berjalan", "batas_percobaan", n.cfg.MaxAttempts)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		n.sendDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-n.wake:
		}
	}
}

// sendDue mengirim semua pesan jatuh tempo (per kelompok 20).
func (n *Notifier) sendDue(ctx context.Context) {
	for ctx.Err() == nil {
		batch, err := n.store.ClaimDueNotifications(ctx, 20, n.cfg.Timeout+time.Minute)
		if err != nil {
			slog.Error("ambil antrian notifikasi gagal", "err", err)
			return
		}
		if len(batch) == 0 {
			return
		}
		for _, m := range batch {
			n.deliver(ctx, m)
		}
	}
}

func (n *Notifier) deliver(ctx context.Context, m domain.Notification) {
	err := n.post(ctx, m.Phone, m.Message)
	if err == nil {
		if err := n.store.MarkNotificationSent(ctx, m.ID); err != nil {
			slog.Error("tandai notifikasi terkirim gagal", "id", m.ID, "err", err)
		}
		return
	}
	attempts := m.Attempts + 1
	var next *time.Time
	if attempts < n.cfg.MaxAttempts {
		t := time.Now().Add(Backoff(attempts))
		next = &t
	}
	slog.Warn("kirim WhatsApp gagal", "id", m.ID, "alert_id", m.AlertID, "percobaan", attempts, "akan_diulang", next != nil, "err", err)
	if err := n.store.MarkNotificationRetry(ctx, m.ID, attempts, err.Error(), next); err != nil {
		slog.Error("tandai notifikasi gagal", "id", m.ID, "err", err)
	}
}

// Backoff = jeda sebelum percobaan berikutnya: 1 menit, 5 menit, lalu 15 menit.
func Backoff(attempts int) time.Duration {
	switch attempts {
	case 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	}
	return 15 * time.Minute
}

// post = kirim ke server WhatsApp (format sistem lama): POST {"number": "62…", "message": "…"}; 2xx = berhasil.
func (n *Notifier) post(ctx context.Context, phone, message string) error {
	body, _ := json.Marshal(map[string]string{"number": phone, "message": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("HTTP %d %s", res.StatusCode, bytes.TrimSpace(b))
	}
	return nil
}
