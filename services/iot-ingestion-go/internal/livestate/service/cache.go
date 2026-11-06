// Package service (fitur livestate) menyimpan kondisi terakhir setiap equipment di MEMORI (ADR 0005).
// Database hanya dibaca sekali saat start; setelah itu setiap data baru cukup DITULIS.
// Bila kelak layanan dijalankan lebih dari satu instance, Cache bisa diganti versi Redis
// yang memenuhi method yang sama, tanpa mengubah service.
package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// Persister = penyimpanan permanen kondisi terakhir (dipenuhi *repository.Repository fitur ini).
type Persister interface {
	LoadLiveStates(ctx context.Context) ([]domain.LiveState, error)
	UpsertLiveState(ctx context.Context, s domain.LiveState) error
}

type Cache struct {
	db  Persister
	now func() time.Time

	mu     sync.RWMutex
	states map[uuid.UUID]domain.LiveState

	locks sync.Map // equipment_id → *sync.Mutex: data satu equipment diproses berurutan
}

func New(db Persister) *Cache {
	return &Cache{db: db, now: time.Now, states: map[uuid.UUID]domain.LiveState{}}
}

// Load mengisi cache dari database. Dipanggil SEKALI saat layanan start.
func (c *Cache) Load(ctx context.Context) (int, error) {
	all, err := c.db.LoadLiveStates(ctx)
	if err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range all {
		c.states[s.EquipmentID] = s
	}
	return len(all), nil
}

// Lock mengunci SATU equipment selama satu event diproses (hitung status muatan → simpan → perbarui
// kondisi terakhir → kejadian), agar data unit yang sama selalu berurutan. Kembalian = pembuka kunci.
func (c *Cache) Lock(equipmentID uuid.UUID) (unlock func()) {
	l := c.lockFor(equipmentID)
	l.Lock()
	return l.Unlock
}

// Apply memperbarui kondisi terakhir sebuah equipment dengan event yang baru tersimpan.
// changed=false bila event lebih lama dari kondisi sekarang (data terlambat).
// Pemanggil yang memproses data bersamaan WAJIB memegang Lock(equipmentID).
func (c *Cache) Apply(ctx context.Context, owner domain.Owner, e *domain.Event, telemetryID int64, receivedAt time.Time) (domain.LiveState, bool, error) {
	var prev *domain.LiveState
	if s, ok := c.Get(owner.EquipmentID); ok {
		prev = &s
	}
	next, changed := domain.NextLiveState(prev, owner, e, telemetryID, receivedAt, c.now())
	if !changed {
		return next, false, nil
	}
	// Tulis ke database DULU; memori baru diubah bila database berhasil, agar keduanya selalu sama.
	if err := c.db.UpsertLiveState(ctx, next); err != nil {
		return next, false, err
	}
	c.mu.Lock()
	c.states[owner.EquipmentID] = next
	c.mu.Unlock()
	return next, true, nil
}

// MarkOffline menandai `offline` setiap equipment yang tidak mengirim data selama ≥ after (dihitung dari jam
// TERIMA server, bukan jam alat). Data berikutnya mengembalikan status normal dengan sendirinya.
// Tidak menulis baris telemetry palsu. Kembalian = kondisi yang berubah (untuk disiarkan ke peta live).
func (c *Cache) MarkOffline(ctx context.Context, now time.Time, after time.Duration) ([]domain.LiveState, error) {
	c.mu.RLock()
	var candidates []uuid.UUID
	for id, s := range c.states {
		if stale(s, now, after) {
			candidates = append(candidates, id)
		}
	}
	c.mu.RUnlock()

	var changed []domain.LiveState
	for _, id := range candidates {
		s, err := c.markOne(ctx, id, now, after)
		if err != nil {
			return changed, err
		}
		if s != nil {
			changed = append(changed, *s)
		}
	}
	return changed, nil
}

func (c *Cache) markOne(ctx context.Context, id uuid.UUID, now time.Time, after time.Duration) (*domain.LiveState, error) {
	unlock := c.Lock(id) // bisa saja data baru masuk di antara pemeriksaan & penguncian
	defer unlock()
	s, ok := c.Get(id)
	if !ok || !stale(s, now, after) {
		return nil, nil
	}
	offline := domain.StatusOffline
	s.EquipmentStatus, s.UpdatedAt = &offline, now
	if err := c.db.UpsertLiveState(ctx, s); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.states[id] = s
	c.mu.Unlock()
	return &s, nil
}

func stale(s domain.LiveState, now time.Time, after time.Duration) bool {
	return now.Sub(s.ReceivedAt) >= after && (s.EquipmentStatus == nil || *s.EquipmentStatus != domain.StatusOffline)
}

// Get mengambil kondisi terakhir satu equipment.
func (c *Cache) Get(equipmentID uuid.UUID) (domain.LiveState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.states[equipmentID]
	return s, ok
}

// List mengambil kondisi terakhir semua equipment sebuah project (nil = semua project, superadmin).
func (c *Cache) List(projectID *uuid.UUID) []domain.LiveState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.LiveState, 0, len(c.states))
	for _, s := range c.states {
		if projectID == nil || (s.ProjectID != nil && *s.ProjectID == *projectID) {
			out = append(out, s)
		}
	}
	return out
}

func (c *Cache) lockFor(equipmentID uuid.UUID) *sync.Mutex {
	l, _ := c.locks.LoadOrStore(equipmentID, &sync.Mutex{})
	return l.(*sync.Mutex)
}
