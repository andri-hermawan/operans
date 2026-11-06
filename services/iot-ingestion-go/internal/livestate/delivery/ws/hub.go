// Package ws = pintu WebSocket fitur livestate (GET /ws/live, socket.go) beserta Hub penyiarnya (hub.go).
//
// Hub menyiarkan perubahan (kondisi equipment, alert, …) ke semua koneksi WebSocket
// yang sedang terbuka, HANYA ke pelanggan project yang bersangkutan (ADR 0005).
//
// Hub ini hidup di memori satu instance. Bila kelak layanan dijalankan lebih dari satu instance,
// Publish cukup diteruskan lewat Redis pub/sub ke Hub di instance lain, tanpa mengubah pemanggilnya.
package ws

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Jenis pesan yang dikirim ke frontend.
const (
	TypeReady          = "ready"           // login WebSocket berhasil
	TypeEquipmentState = "equipment_state" // kondisi terakhir satu equipment berubah
	TypeGeofenceEvent  = "geofence_event"  // unit masuk / keluar zona (Fase C2)
	TypeFuelEvent      = "fuel_event"      // isi / turun BBM (Fase C3)
	TypeAlert          = "alert"           // alert dibuka / ditutup (Fase C3–C4)
)

// Envelope = bentuk SEMUA pesan WebSocket: {"type": "...", "data": {...}}.
type Envelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

var ErrFull = errors.New("jumlah koneksi live sudah maksimum")

// Alasan pelanggan diputus oleh hub.
const (
	ReasonSlow     = "koneksi terlalu lambat menerima data"
	ReasonShutdown = "server berhenti"
)

type Hub struct {
	maxSubs int
	bufSize int

	mu     sync.RWMutex
	subs   map[*Subscriber]struct{}
	closed bool
}

// NewHub: maxSubs = batas koneksi bersamaan; bufSize = antrean pesan per koneksi.
func NewHub(maxSubs, bufSize int) *Hub {
	return &Hub{maxSubs: maxSubs, bufSize: bufSize, subs: map[*Subscriber]struct{}{}}
}

// Subscriber = satu koneksi WebSocket yang sedang mendengarkan.
type Subscriber struct {
	scope  *uuid.UUID  // nil = semua project (superadmin)
	ch     chan []byte // antrean pesan siap kirim
	done   chan struct{}
	once   sync.Once
	reason string
}

// C = pesan yang harus dikirim ke klien.
func (s *Subscriber) C() <-chan []byte { return s.ch }

// Done tertutup bila hub memutus pelanggan ini (lihat Reason).
func (s *Subscriber) Done() <-chan struct{} { return s.done }
func (s *Subscriber) Reason() string        { return s.reason }

func (s *Subscriber) stop(reason string) {
	s.once.Do(func() {
		s.reason = reason
		close(s.done)
	})
}

// wants: pesan untuk project p dikirim ke pelanggan project p, dan ke superadmin.
func (s *Subscriber) wants(p *uuid.UUID) bool {
	return s.scope == nil || (p != nil && *p == *s.scope)
}

// Subscribe mendaftarkan koneksi baru dengan cakupan project dari token.
func (h *Hub) Subscribe(scope *uuid.UUID) (*Subscriber, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || len(h.subs) >= h.maxSubs {
		return nil, ErrFull
	}
	s := &Subscriber{scope: scope, ch: make(chan []byte, h.bufSize), done: make(chan struct{})}
	h.subs[s] = struct{}{}
	return s, nil
}

// Unsubscribe dipanggil saat koneksi ditutup.
func (h *Hub) Unsubscribe(s *Subscriber) {
	h.mu.Lock()
	delete(h.subs, s)
	h.mu.Unlock()
}

// Publish mengirim satu pesan ke semua pelanggan project projectID (nil = hanya superadmin).
// Tidak pernah menunggu: pelanggan yang antreannya penuh diputus agar tidak memperlambat yang lain.
func (h *Hub) Publish(projectID *uuid.UUID, msgType string, data any) {
	b, err := json.Marshal(Envelope{Type: msgType, Data: data})
	if err != nil {
		slog.Error("live: pesan tidak bisa diubah ke JSON", "type", msgType, "err", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs {
		if !s.wants(projectID) {
			continue
		}
		select {
		case s.ch <- b:
		default:
			s.stop(ReasonSlow)
		}
	}
}

// Close memutus semua pelanggan (dipanggil saat server berhenti).
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for s := range h.subs {
		s.stop(ReasonShutdown)
	}
}

// Count = jumlah koneksi live saat ini.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
