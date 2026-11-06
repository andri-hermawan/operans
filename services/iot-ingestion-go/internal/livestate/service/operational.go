package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"simha/iot-ingestion/internal/ingest/domain"
)

// Operational = keadaan operasional sebuah unit menurut data induk.
type Operational struct {
	BreakdownKnown       bool // false = unit belum punya catatan breakdown → nilai yang ada dibiarkan
	IsBreakdown          bool
	BreakdownDescription *string // mis. "AC HOT"; hanya selama masih breakdown
	OperatorKnown        bool    // false = shift belum diketahui → operator yang ada dibiarkan
	Operator             *string
}

// OperationalResolver = sumber keadaan operasional sebuah unit pada waktu t (dipenuhi adaptor di cmd/iot atas
// *masterdata.Cache). ok=false bila project/unit belum diketahui → semuanya dibiarkan.
type OperationalResolver interface {
	OperationalAt(projectID, equipmentID *uuid.UUID, t time.Time) (op Operational, ok bool)
}

// RefreshOperational menyamakan is_breakdown, breakdown_description, dan operator SEMUA unit dengan data induk saat ini,
// tanpa menunggu data telemetry berikutnya (unit offline pun ikut). Dipanggil saat breakdown / operator diinput,
// diubah, diunggah, atau dihapus, dan berkala untuk operator yang berganti menurut shift. Status breakdown MENEMPEL:
// ia hanya berubah bila catatan terbaru yang diinput berubah (mis. menjadi ready), bukan karena waktu.
// Yang berubah ditulis ke iot.equipment_live_states dan dikembalikan (untuk disiarkan ke peta live).
func (c *Cache) RefreshOperational(ctx context.Context, r OperationalResolver, now time.Time) ([]domain.LiveState, error) {
	c.mu.RLock()
	ids := make([]uuid.UUID, 0, len(c.states))
	for id := range c.states {
		ids = append(ids, id)
	}
	c.mu.RUnlock()

	var changed []domain.LiveState
	for _, id := range ids {
		s, err := c.refreshOne(ctx, r, id, now)
		if err != nil {
			return changed, err
		}
		if s != nil {
			changed = append(changed, *s)
		}
	}
	return changed, nil
}

func (c *Cache) refreshOne(ctx context.Context, r OperationalResolver, id uuid.UUID, now time.Time) (*domain.LiveState, error) {
	unlock := c.Lock(id) // data baru unit ini bisa masuk bersamaan: satu per satu
	defer unlock()
	s, ok := c.Get(id)
	if !ok {
		return nil, nil
	}
	op, known := r.OperationalAt(s.ProjectID, &id, now)
	if !known {
		return nil, nil
	}
	next := s
	if op.BreakdownKnown {
		next.IsBreakdown, next.BreakdownDescription = &op.IsBreakdown, op.BreakdownDescription
	}
	if op.OperatorKnown {
		next.OperatorName = op.Operator
	}
	if sameBool(s.IsBreakdown, next.IsBreakdown) && sameString(s.BreakdownDescription, next.BreakdownDescription) && sameString(s.OperatorName, next.OperatorName) {
		return nil, nil
	}
	next.UpdatedAt = now
	// Tulis ke database DULU; memori baru diubah bila database berhasil, agar keduanya selalu sama.
	if err := c.db.UpsertLiveState(ctx, next); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.states[id] = next
	c.mu.Unlock()
	return &next, nil
}

func sameBool(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// WatchOperational menjaga kondisi terakhir unit tetap sama dengan data induk: segera saat ada sinyal di `trigger`
// (data induk baru dimuat ulang karena input/ubah/unggah/hapus), dan tiap `every` untuk operator yang berganti menurut
// shift. Status breakdown TIDAK berubah karena waktu. Yang berubah disiarkan ke peta live.
func WatchOperational(ctx context.Context, states *Cache, r OperationalResolver, hub Broadcaster, trigger <-chan struct{}, every time.Duration) {
	slog.Info("pemeriksa breakdown & operator berjalan", "berkala", every.String())
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		changed, err := states.RefreshOperational(ctx, r, time.Now())
		if err != nil {
			slog.Error("perbarui breakdown/operator kondisi terakhir gagal", "err", err)
		}
		for _, s := range changed {
			hub.Publish(s.ProjectID, msgEquipmentState, s)
		}
		if len(changed) > 0 {
			slog.Info("breakdown/operator kondisi terakhir diperbarui", "jumlah", len(changed))
		}
		select {
		case <-ctx.Done():
			return
		case <-trigger:
		case <-t.C:
		}
	}
}
