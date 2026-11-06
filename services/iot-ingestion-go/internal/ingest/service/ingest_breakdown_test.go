package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/ingest/repository"
)

// stickyStore mencatat event yang benar-benar disimpan.
type stickyStore struct {
	*fakeStore
	last domain.Event
}

func (p *stickyStore) InsertEvent(ctx context.Context, d *devicedomain.Device, e *domain.Event) (repository.Saved, bool, error) {
	p.last = *e
	return p.fakeStore.InsertEvent(ctx, d, e)
}

// stickyLive = kondisi terakhir palsu yang SUDAH berisi status breakdown (diinput tim, "AC HOT").
type stickyLive struct{ fakeLive }

func (stickyLive) Get(uuid.UUID) (domain.LiveState, bool) {
	yes, desc := true, "AC HOT"
	return domain.LiveState{IsBreakdown: &yes, BreakdownDescription: &desc, EventTime: now.Add(-time.Hour)}, true
}

// Unit tanpa catatan breakdown di data induk: data baru MENERUSKAN status breakdown yang sedang berlaku
// (tidak ditimpa false hanya karena unit bergerak / shift berganti).
func TestHandleBreakdownMenempel(t *testing.T) {
	store := &stickyStore{fakeStore: &fakeStore{saved: map[string]bool{}}}
	s := NewIngest(store, &stickyLive{}, nil)
	s.now = func() time.Time { return now }
	e := event("sticky-1")
	spd := 32.0
	e.SpeedKph = &spd // unit ternyata bergerak
	if _, err := s.Handle(context.Background(), dev, e); err != nil {
		t.Fatal(err)
	}
	got := store.last
	if got.IsBreakdown == nil || !*got.IsBreakdown || got.BreakdownDescription == nil || *got.BreakdownDescription != "AC HOT" {
		t.Fatalf("breakdown=%v desc=%v", got.IsBreakdown, got.BreakdownDescription)
	}
}
