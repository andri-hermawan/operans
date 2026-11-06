# core-api-go

Layanan data induk SIMHA (port **3346**): login & token, user, company/project/shift, equipment, input harian
(operator, breakdown, weighbridge, upload Excel), peta area, aturan alert, laporan & jadwal laporan.

```bash
go run ./cmd/api
```

Pengaturan dibaca otomatis dari `.env` (contoh: `.env.example`). Cara menjalankan: `../../README.md`.

## Peta folder

```
core-api-go/
├── cmd/api/main.go     ← mulai membaca dari sini: semua fitur dirakit & route didaftarkan
├── internal/           ← kode aplikasi (satu folder = satu fitur/menu)
│   ├── shared/         ← bagian teknis yang dipakai banyak fitur
│   └── <fitur>/        ← lihat daftar di bawah
├── pkg/hash/           ← hash password (argon2id + bcrypt sistem lama), tidak bergantung SIMHA
├── migrations/         ← perubahan struktur database (golang-migrate) — JANGAN dinomori ulang
├── db/                 ← seed (data awal, project UJI) & skrip admin database
├── tools/              ← program bantu (create-user, keygen, import-fms)
├── keys/               ← kunci privat JWT (RAHASIA, tidak masuk Git)
└── bin/                ← hasil build (tidak masuk Git)
```

## Daftar fitur (`internal/`)

| Folder | Menu / gunanya | URL utama |
|---|---|---|
| `auth` | login, perpanjang sesi, ganti project, logout, profil, ganti password, kunci publik | `/auth/*`, `/.well-known/jwks.json` |
| `user` | System Data → User (+ reset password, anggota project) | `/api/v1/users`, `/projects/:id/members` |
| `company` | System Data → Company | `/api/v1/companies` |
| `project` | System Data → Project | `/api/v1/projects` |
| `shift` | System Data → Shift & pilihan shift | `/api/v1/shifts`, `/shifts/manage` |
| `equipment` | System Data → Equipment | `/api/v1/equipment` |
| `activation` | Record/Upload → Equipment Activation | `/api/v1/equipment-activations` |
| `fuelcalibration` | System Data → Fuel Calibration | `/api/v1/fuel-calibrations`, `/equipment/:id/fuel-calibration` |
| `reference` | pilihan kategori breakdown & repair status | `/api/v1/reference-values` |
| `operator` | Record/Upload → Setting Operator | `/api/v1/operator-assignments` |
| `breakdown` | Record/Upload → Status Breakdown | `/api/v1/equipment-breakdowns` |
| `weighbridge` | Record/Upload → Weighbridge | `/api/v1/weighbridge-tickets` |
| `importexcel` | tombol Upload Excel (template, pratinjau, simpan) | `/api/v1/imports/:type` |
| `maparea` | System Data → Map Area (GeoJSON, zona) | `/api/v1/geo/*` |
| `alerttype` | System Data → Alert Category | `/api/v1/alert-types` |
| `alertrule` | System Data → Alert Rule | `/api/v1/alert-rules` |
| `report` | Report: Ritase, Payload, Produksi, Breakdown, PA | `/api/v1/reports/*` |
| `reportschedule` | System Data → Report Schedule (PDF + email) | `/api/v1/report-schedules` |

`internal/shared/`: `config` (.env), `database` (koneksi pgx), `middleware` (cek token, request id, log),
`response` (format JSON & galat), `access` (siapa pemanggil & haknya), `apperror` (galat bersama),
`token` (JWT RS256), `validate` (pemeriksa kecil), `excel` (pembaca Excel/CSV), `dailyinput` (aturan tanggal/shift/unit
yang dipakai operator, breakdown, weighbridge & upload Excel).

## Isi satu fitur (semua fitur sama)

```
internal/weighbridge/
├── domain/entity.go          struct data + aturan murni
├── repository/repository.go  SQL ke tabel fitur ini (pgx)
├── service/service.go        validasi, hak akses, panggil repository
└── delivery/http/handler.go  route + handler: baca request → panggil service → kirim JSON
```

Alur satu request: **handler → service → repository → database**.

## Cara membaca alur sebuah fitur (contoh: simpan tiket weighbridge)

1. `cmd/api/main.go` — cari `weighbridge`: repository → service → handler dibuat, lalu `Register(api)`.
2. `internal/weighbridge/delivery/http/handler.go` — `Register` menunjukkan `POST /weighbridge-tickets` → `h.Set`.
3. `internal/weighbridge/service/service.go` — `Set`: cek hak, cek tanggal/shift/unit, hitung net.
4. `internal/weighbridge/repository/repository.go` — SQL `INSERT … ON CONFLICT` ke `core.weighbridge_tickets`.

## Membuat fitur baru

1. Salin folder fitur yang paling mirip (mis. `company/`), ganti nama paket & isinya.
2. Tulis struct di `domain`, SQL di `repository`, aturan & hak akses di `service`, route di `delivery/http`.
3. Butuh data fitur lain? Tulis interface kecil di `service` Anda (mis. `ShiftLister`), lalu isi di `main.go`
   dengan service fitur itu. **Jangan** memanggil repository fitur lain.
4. Rakit di `cmd/api/main.go` dan panggil `Register`.
5. Tabel baru → buat migration baru dengan nomor berikutnya di `migrations/`.

## Pemeriksaan

```bash
go build ./... && go vet ./... && go test ./...
```

