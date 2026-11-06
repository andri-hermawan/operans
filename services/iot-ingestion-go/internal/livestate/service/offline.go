package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Broadcaster = penyiar perubahan ke WebSocket (dipenuhi *ws.Hub fitur ini).
type Broadcaster interface {
	Publish(projectID *uuid.UUID, msgType string, data any)
}

// msgEquipmentState = jenis pesan siaran (sama dengan ws.TypeEquipmentState; ditulis ulang agar service
// tidak bergantung pada delivery).
const msgEquipmentState = "equipment_state"

// WatchOffline memeriksa tiap menit unit yang berhenti mengirim data, menandainya offline, dan menyiarkannya.
func WatchOffline(ctx context.Context, states *Cache, hub Broadcaster, after time.Duration) {
	slog.Info("pemeriksa offline berjalan", "batas", after.String())
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		changed, err := states.MarkOffline(ctx, time.Now(), after)
		if err != nil {
			slog.Error("tandai offline gagal", "err", err)
		}
		for _, s := range changed {
			hub.Publish(s.ProjectID, msgEquipmentState, s)
		}
		if len(changed) > 0 {
			slog.Info("unit ditandai offline", "jumlah", len(changed))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
