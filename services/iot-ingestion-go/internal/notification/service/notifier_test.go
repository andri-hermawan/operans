package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	ingestdomain "simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/notification/domain"
	"simha/iot-ingestion/internal/shared/masterdata"
)

// memStore = antrian palsu di memori.
type memStore struct {
	mu    sync.Mutex
	rows  []row
	limit int
}

type row struct {
	domain.Notification
	status string
	next   time.Time
	err    string
}

func (m *memStore) EnqueueNotifications(_ context.Context, ns []domain.Notification) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, n := range ns {
		n.ID = int64(len(m.rows) + 1)
		m.rows = append(m.rows, row{Notification: n, status: domain.NotifyPending})
	}
	return len(ns), nil
}

func (m *memStore) ClaimDueNotifications(_ context.Context, limit int, lease time.Duration) ([]domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Notification
	for i := range m.rows {
		r := &m.rows[i]
		if r.status == domain.NotifyPending && !r.next.After(time.Now()) && len(out) < limit {
			r.next = time.Now().Add(lease)
			out = append(out, r.Notification)
		}
	}
	return out, nil
}

func (m *memStore) MarkNotificationSent(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[id-1].status, m.rows[id-1].Attempts = domain.NotifySent, m.rows[id-1].Attempts+1
	return nil
}

func (m *memStore) MarkNotificationRetry(_ context.Context, id int64, attempts int, msg string, next *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := &m.rows[id-1]
	r.Attempts, r.err = attempts, msg
	if next == nil {
		r.status = domain.NotifyFailed
	} else {
		r.next = time.Now() // uji: langsung jatuh tempo lagi (tanpa menunggu jeda sungguhan)
	}
	return nil
}

type dir struct{ rs []masterdata.Recipient }

func (d dir) Recipients(ids []uuid.UUID) []masterdata.Recipient { return d.rs }
func (d dir) Label(_, _ *uuid.UUID) (string, string, *time.Location) {
	return "DT10300", "Coal Hauling Road", time.FixedZone("WIB", 7*3600)
}

var (
	budi  = masterdata.Recipient{UserID: uuid.New(), Name: "Budi", Phone: "0812-3456-7890"}
	andi  = masterdata.Recipient{UserID: uuid.New(), Name: "Andi", Phone: "+62 812 3456 7890"} // nomor sama dengan Budi
	citra = masterdata.Recipient{UserID: uuid.New(), Name: "Citra", Phone: "85211112222"}
)

func alert() ingestdomain.Alert {
	speed, lat, lon, th := 62.0, -3.6, 103.85, 50.0
	seg, shift := "Km 23+000 - 24+000", "SHIFT_1"
	return ingestdomain.Alert{ID: 7, AlertType: ingestdomain.AlertOverspeed, StartedAt: time.Date(2026, 10, 23, 1, 30, 0, 0, time.UTC),
		ShiftCode: &shift, ThresholdValue: &th, Recipients: []uuid.UUID{budi.UserID, andi.UserID, citra.UserID},
		Snapshot: ingestdomain.Snapshot{SpeedKph: &speed, Latitude: &lat, Longitude: &lon, SegmentName: &seg}}
}

func TestAlertOpenedDanKirim(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]string
	wa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		mu.Lock()
		got = append(got, b)
		mu.Unlock()
	}))
	defer wa.Close()

	st := &memStore{}
	n := New(Config{URL: wa.URL}, st, dir{[]masterdata.Recipient{budi, andi, citra}})
	n.AlertOpened(context.Background(), alert())
	if len(st.rows) != 2 || st.rows[0].Phone != "6281234567890" || st.rows[1].Phone != "6285211112222" {
		t.Fatalf("antrian (nomor sama hanya sekali, dinormalkan 62…): %+v", st.rows)
	}
	n.sendDue(context.Background())
	if len(got) != 2 || st.rows[0].status != domain.NotifySent || st.rows[1].status != domain.NotifySent {
		t.Fatalf("terkirim %d, status %s/%s", len(got), st.rows[0].status, st.rows[1].status)
	}
	msg := got[0]["message"]
	for _, want := range []string{"SIMHA ALERT – Overspeed", "Equipment : DT10300", "Coal Hauling Road / Km 23+000 - 24+000",
		"Speed     : 62 km/h", "Ambang    : 50 km/h", "Waktu     : 23-10-2026 08:30 WIB (SHIFT_1)", "maps.google.com/?q=-3.6,103.85"} {
		if !strings.Contains(msg, want) {
			t.Errorf("pesan tidak memuat %q:\n%s", want, msg)
		}
	}
}

func TestCobaUlangLaluGagal(t *testing.T) {
	calls := 0
	wa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "server WA sibuk", http.StatusServiceUnavailable)
	}))
	defer wa.Close()
	st := &memStore{}
	n := New(Config{URL: wa.URL, MaxAttempts: 3}, st, dir{[]masterdata.Recipient{budi}})
	n.AlertOpened(context.Background(), alert())
	for i := 0; i < 5; i++ {
		n.sendDue(context.Background())
	}
	r := st.rows[0]
	if calls != 3 || r.status != domain.NotifyFailed || r.Attempts != 3 || !strings.Contains(r.err, "HTTP 503") {
		t.Fatalf("calls=%d %+v", calls, r)
	}
	if Backoff(1) != time.Minute || Backoff(2) != 5*time.Minute || Backoff(3) != 15*time.Minute {
		t.Fatal("jeda coba ulang berubah")
	}
}

func TestNonaktif(t *testing.T) {
	st := &memStore{}
	n := New(Config{URL: ""}, st, dir{[]masterdata.Recipient{budi}})
	n.AlertOpened(context.Background(), alert())
	if n.Enabled() || len(st.rows) != 0 {
		t.Fatal("WHATSAPP_API_URL kosong: tidak boleh ada antrian")
	}
	n.Run(context.Background(), time.Millisecond) // langsung kembali, tidak berputar
}

func TestNormalizePhone(t *testing.T) {
	for in, want := range map[string]string{"0812-3456-7890": "6281234567890", "+62 812 3456 7890": "6281234567890",
		"81234567890": "6281234567890", "6281234567890": "6281234567890", "": "", "abc": ""} {
		if got := domain.NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q; mau %q", in, got, want)
		}
	}
}
