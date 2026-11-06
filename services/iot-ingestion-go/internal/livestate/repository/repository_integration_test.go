//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	devicedomain "simha/iot-ingestion/internal/device/domain"
	devicerepo "simha/iot-ingestion/internal/device/repository"
	"simha/iot-ingestion/internal/ingest/domain"
	"simha/iot-ingestion/internal/shared/database"
)

// Kondisi terakhir tersimpan, terbaca kembali, dan data yang lebih lama TIDAK menimpanya
// (pengaman kedua di SQL, selain pengaman di memori).
func TestLiveStateSimpanMuatDanPengaman(t *testing.T) {
	ctx := context.Background()
	repo, dev, _ := setup(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	lat, lon, speed := -3.47, 103.849, 21.0
	status := domain.StatusRunning
	opDate, shift := "2026-10-22", "SHIFT_2"

	baru := domain.LiveState{
		EquipmentID: *dev.EquipmentID, ProjectID: dev.ProjectID, DeviceID: dev.ID,
		TelemetryEventID: 1_000_000, EventTime: now, ReceivedAt: now, UpdatedAt: now,
		OperationalDate: &opDate, ShiftCode: &shift,
		Latitude: &lat, Longitude: &lon, SpeedKph: &speed, EquipmentStatus: &status,
		AlertCount: 2,
	}
	if err := repo.UpsertLiveState(ctx, baru); err != nil {
		t.Fatalf("simpan: %v", err)
	}

	lama := baru
	lama.TelemetryEventID = 999
	lama.EventTime = now.Add(-time.Hour)
	if err := repo.UpsertLiveState(ctx, lama); err != nil {
		t.Fatalf("simpan data lama: %v", err)
	}

	all, err := repo.LoadLiveStates(ctx)
	if err != nil {
		t.Fatalf("muat: %v", err)
	}
	for _, s := range all {
		if s.EquipmentID != *dev.EquipmentID {
			continue
		}
		switch {
		case s.TelemetryEventID != 1_000_000:
			t.Fatalf("data lama menimpa data baru: telemetry_event_id = %d", s.TelemetryEventID)
		case s.Latitude == nil || *s.Latitude != lat || *s.Longitude != lon:
			t.Fatalf("posisi tidak cocok: %v,%v", s.Latitude, s.Longitude)
		case s.OperationalDate == nil || *s.OperationalDate != opDate || *s.ShiftCode != shift:
			t.Fatalf("tanggal operasional/shift tidak cocok: %v %v", s.OperationalDate, s.ShiftCode)
		case *s.EquipmentStatus != status || s.AlertCount != 2:
			t.Fatalf("status/alert_count tidak cocok: %v %d", *s.EquipmentStatus, s.AlertCount)
		}
		t.Log("OK: kondisi terakhir tersimpan, terbaca kembali, dan tidak tertimpa data lama")
		return
	}
	t.Fatal("kondisi terakhir equipment uji tidak ditemukan")
}

// Jalankan: go test -tags integration ./internal/livestate/repository/ -v
// Butuh DATABASE_URL (akun iot_ingest) dan alat simulator yang sudah terdaftar.

// setup membuka koneksi dan mengambil alat uji (bawaan: simulator 000000000000001, BUKAN alat asli).
func setup(t *testing.T) (*Repository, *devicedomain.Device, string) {
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
	return New(pool), dev, imei
}
