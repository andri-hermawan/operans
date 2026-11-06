# iot-ingestion-go

Layanan IoT SIMHA: menerima data alat GPS Teltonika (**TCP 3347**) & simulator (**HTTP 5550**), mengolahnya
(zona, status, muatan, fuel, alert, geofence), menyimpan ke schema `iot`, menyiarkan ke peta live (WebSocket), dan
melayani API baca untuk halaman pemantauan.

```bash
go run ./cmd/iot
```

Pengaturan dibaca otomatis dari `.env` (contoh: `.env.example`). Cara menjalankan: `../../README.md`.

## Peta folder

```
iot-ingestion-go/
├── cmd/iot/main.go     ← mulai membaca dari sini: semua fitur dirakit, HTTP + TCP + WebSocket dijalankan
├── internal/
│   ├── shared/         ← config, database, middleware (token & request id), response, masterdata
│   └── <fitur>/        ← lihat daftar di bawah
├── pkg/teltonika/      ← pembaca protokol Teltonika Codec 8/8E (tidak bergantung SIMHA)
├── migrations/         ← perubahan struktur schema iot — JANGAN dinomori ulang
├── tools/              ← simulator-dt03, dt07 & ex01 (peniru alat GPS untuk uji)
└── docs/               ← panduan khusus IoT (kosongkan-dan-simulasi.md)
```

## Daftar fitur (`internal/`)

| Folder | Gunanya | Pintu masuk |
|---|---|---|
| `ingest` | terima & olah data alat (pipeline). `domain/` berisi semua aturan: status, muatan, zona, fuel, alert, geofence, shift | TCP 9001 (`delivery/tcp`), `POST /api/v1/ingest/events` (`delivery/http`) |
| `livestate` | kondisi terakhir tiap unit di memori, WebSocket peta live, penanda offline | `/ws/live` (`delivery/ws`), `/api/v1/equipment-live-states` |
| `telemetry` | Position/Fuel History, Speed Heatmap, Activity Summary, data terakhir & riwayat | `/api/v1/telemetry/*`, `/activity-summary`, `/equipment/:id/latest`, `/history` |
| `alert` | daftar alert (Abnormal Alerts, Distribution Map) | `/api/v1/alerts` |
| `geofence` | kejadian masuk/keluar area (Geofencing) | `/api/v1/geofence-events` |
| `fuel` | kejadian isi/turun BBM | `/api/v1/fuel-events` |
| `notification` | kirim WhatsApp dari antrian (berjalan di belakang, tanpa URL) | — |
| `device` | System Data → Device, API key, status alat | `/api/v1/devices` |

`internal/shared/masterdata` = salinan data induk core (unit, shift, operator, zona, kalibrasi, aturan alert) di memori,
diperbarui otomatis saat core berubah (LISTEN/NOTIFY).

## Isi satu fitur

```
internal/alert/
├── domain/entity.go
├── repository/repository.go
├── service/service.go
└── delivery/http/handler.go
```

Alur: **handler → service → repository → database**. Pengecualian yang disengaja (ada komentarnya di kode):
tabel `iot.telemetry_events`, `iot.alerts`, `iot.geofence_events`, `iot.fuel_events` **ditulis** oleh `ingest`
(pipeline) dan **dibaca** oleh fitur telemetry/alert/geofence/fuel/device.

## Alur satu data dari alat

1. `ingest/delivery/tcp/listener.go` — Teltonika login dengan IMEI, paket dibaca `pkg/teltonika`, diubah ke `Event`.
2. `ingest/service/ingest.go` — `Handle`: validasi → pengayaan (shift, operator, zona dari masterdata) → aturan
   (status, muatan, fuel, alert, geofence di `ingest/domain`) → simpan (`ingest/repository`).
3. `livestate` memperbarui kondisi terakhir → `delivery/ws` menyiarkan ke browser; `notification` mengantre WhatsApp.
4. Alat menerima ACK hanya setelah data tersimpan.

## Pemeriksaan

```bash
go build ./... && go vet ./... && go test ./...
```

Script: `tools\simulasi-dt03.ps1` (ulang simulasi DT03), `tools\uji-f3.ps1` (endpoint pemantauan).
