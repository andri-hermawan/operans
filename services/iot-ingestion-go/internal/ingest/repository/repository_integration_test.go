//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	devicerepo "simha/iot-ingestion/internal/device/repository"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/shared/database"
	telemetrydomain "simha/iot-ingestion/internal/telemetry/domain"
	telemetryrepo "simha/iot-ingestion/internal/telemetry/repository"
)

// Jalankan: go test -tags integration ./internal/ingest/repository/ -v
// Butuh DATABASE_URL (akun iot_ingest) dan alat simulator yang sudah terdaftar.
// Data uji ditandai metadata.source = "integration-test".

// setup membuka koneksi dan mengambil alat uji (bawaan: simulator 000000000000001, BUKAN alat asli).
// Data dibaca kembali lewat repository fitur telemetry (pembaca iot.telemetry_events).
func setup(t *testing.T) (*Repository, *telemetryrepo.Repository, *devicedomain.Device, string) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL kosong: muat .env dulu")
	}
	imei := os.Getenv("TEST_DEVICE_IMEI")
	if imei == "" {
		imei = "000000000000001"
	}
	pool, err := database.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("koneksi: %v", err)
	}
	t.Cleanup(pool.Close)

	dev, err := devicerepo.New(pool).FindDeviceByIMEI(context.Background(), imei)
	if err != nil {
		t.Fatalf("cari device: %v", err)
	}
	if dev == nil {
		t.Fatalf("IMEI %s belum terdaftar / tidak aktif di iot.devices", imei)
	}
	if dev.EquipmentID == nil || dev.ProjectID == nil {
		t.Fatalf("alat uji %s harus punya equipment_id dan project_id", imei)
	}
	return New(pool), telemetryrepo.New(pool), dev, imei
}

func TestInsertEventIdempotent(t *testing.T) {
	ctx := context.Background()
	repo, reader, dev, imei := setup(t)

	// 2) Event uji yang unik per run (PostgreSQL menyimpan sampai mikrodetik)
	now := time.Now().UTC().Truncate(time.Microsecond)
	lat, lon, speed, raw := -1.2654, 116.8312, 32.5, 2048.0
	e := &domain.Event{
		EventID:   fmt.Sprintf("itest-%d", now.UnixNano()),
		IMEI:      imei,
		EventTime: now,
		Position:  domain.Position{Latitude: &lat, Longitude: &lon, SpeedKph: &speed},
		Fuel:      domain.Fuel{FuelLevelRaw: &raw},
		Metadata:  map[string]any{"source": "integration-test"},
	}

	// 3) Kirim pertama → tersimpan
	saved, inserted, err := repo.InsertEvent(ctx, dev, e)
	if err != nil {
		t.Fatalf("insert pertama: %v", err)
	}
	if !inserted || saved.ID == 0 || saved.ReceivedAt.IsZero() {
		t.Fatalf("insert pertama seharusnya tersimpan dan mengembalikan id: %+v", saved)
	}

	// 4) Kirim ulang event yang SAMA → diabaikan, tanpa error
	_, inserted, err = repo.InsertEvent(ctx, dev, e)
	if err != nil {
		t.Fatalf("insert kedua: %v", err)
	}
	if inserted {
		t.Fatal("insert kedua seharusnya dianggap duplikat")
	}

	// 5) Baca kembali dan cocokkan nilainya
	rows, err := reader.HistoryByEquipment(ctx, dev.ProjectID, *dev.EquipmentID, now.Add(-time.Second), now.Add(time.Second), 50)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	for _, x := range rows {
		if x.EventID != e.EventID {
			continue
		}
		if x.Latitude == nil || *x.Latitude != lat || *x.Longitude != lon {
			t.Fatalf("posisi tidak cocok: %v,%v", x.Latitude, x.Longitude)
		}
		if x.SpeedKph == nil || *x.SpeedKph != speed || x.FuelLevelRaw == nil || *x.FuelLevelRaw != raw {
			t.Fatal("speed / fuel_level_raw tidak cocok")
		}
		if x.FuelPercent != nil {
			t.Fatal("fuel_percent tidak dikirim, seharusnya tetap NULL (nil), bukan 0")
		}
		t.Logf("OK: event %s tersimpan sekali dan terbaca kembali", e.EventID)
		return
	}
	t.Fatal("event uji tidak ditemukan di history")
}

// Data terlambat (alat offline 2 jam) tetap tersimpan; project lain tidak bisa melihatnya.
func TestDataTerlambatDanIsolasiProject(t *testing.T) {
	ctx := context.Background()
	repo, reader, dev, imei := setup(t)

	late := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)
	lat, lon := -3.47, 103.849
	e := &domain.Event{
		EventID:   fmt.Sprintf("itest-late-%d", late.UnixNano()),
		IMEI:      imei,
		EventTime: late,
		Position:  domain.Position{Latitude: &lat, Longitude: &lon},
		Metadata:  map[string]any{"source": "integration-test"},
	}
	if _, inserted, err := repo.InsertEvent(ctx, dev, e); err != nil || !inserted {
		t.Fatalf("data terlambat seharusnya tersimpan: inserted=%v err=%v", inserted, err)
	}

	window := func(projectID *uuid.UUID) []telemetrydomain.TelemetryRow {
		rows, err := reader.HistoryByEquipment(ctx, projectID, *dev.EquipmentID, late.Add(-time.Second), late.Add(time.Second), 50)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	find := func(rows []telemetrydomain.TelemetryRow) *telemetrydomain.TelemetryRow {
		for i := range rows {
			if rows[i].EventID == e.EventID {
				return &rows[i]
			}
		}
		return nil
	}

	// Project sendiri: terlihat, dan waktu terima (created_at) > waktu kejadian (event_time)
	own := find(window(dev.ProjectID))
	if own == nil {
		t.Fatal("project sendiri seharusnya melihat data ini")
	}
	if !own.CreatedAt.After(own.EventTime) {
		t.Fatalf("created_at (%s) seharusnya setelah event_time (%s)", own.CreatedAt, own.EventTime)
	}

	// Project lain: tidak terlihat
	other := uuid.New()
	if find(window(&other)) != nil {
		t.Fatal("project lain TIDAK boleh melihat data ini")
	}

	// Superadmin (nil): terlihat
	if find(window(nil)) == nil {
		t.Fatal("superadmin seharusnya melihat semua project")
	}
	t.Logf("OK: data terlambat %s tersimpan, terlambat %s, dan terisolasi per project",
		e.EventID, own.CreatedAt.Sub(own.EventTime).Round(time.Second))
}
