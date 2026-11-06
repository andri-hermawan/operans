# SIMHA — Mining & Hauling Platform

Platform pemantauan dump truck tambang: posisi live, status muatan, fuel, alert, input harian, dan laporan.

## Isi repositori

```
operans/
├── README.md              ← Anda di sini
├── go.work                ← supaya VS Code mengenali kedua service Go sekaligus
├── docs/
│   ├── PROGRES.md         ← satu-satunya catatan progres
│   └── adr/               ← catatan keputusan (ADR 0001–0015)
├── services/
│   ├── core-api-go/       ← login, user, master data, input harian & rencana (Record / Upload), peta area, laporan (port 3346)
│   ├── iot-ingestion-go/  ← terima data alat GPS, peta live, riwayat, alert (HTTP 3347 & TCP alat GPS 5550)
│   └── ai-brain-py/       ← AI Brain (Python/FastAPI): pustaka SOP, pencarian (port 3348)
└── frontend/              ← aplikasi web React (port 8881)
```

## Fitur utama

- **Monitoring:** peta live, detail unit, alert, fuel, dashboard.
- **Record / Upload** (input manual atau impor Excel/CSV, hak akses per project):
  - *Transaction:* Setting Operator, Equipment Activation, Status Breakdown, Weighbridge, Stock Emplacement, Time Sheet, Ritase, WH & Notification, Rain & Slippery.
  - *Plan:* Plan Yearly, Plan Monthly.
- **Laporan:** harian, per shift, executive; email terjadwal.
- **Dokumen SOP:** pustaka & pencarian dokumen (AI Brain; model bahasa/Ollama belum dipasang).
- **System Data:** user, project, perusahaan, equipment, shift, peta area, aturan alert.

## Cara kerja singkat

```
Alat GPS Teltonika ──TCP──▶ iot-ingestion-go ──▶ database simha (schema iot)
                                  │  ▲
                     WebSocket    │  │ data induk (unit, shift, peta, aturan) dibaca dari schema core/geo
                     & API baca   ▼  │
Browser ◀── frontend (React) ──▶ core-api-go ──▶ database simha (schema auth, core, geo)
```

- **core-api-go** memegang login (token RS256) dan data induk. Token yang sama dipakai untuk memanggil IoT.
- **iot-ingestion-go** mengolah setiap data alat (zona, status, muatan, fuel, alert) lalu menyiarkannya ke peta live.

## Struktur kode service (ADR 0011)

Setiap service Go memakai pola yang sama: **satu fitur (menu) = satu folder** di `internal/`, berisi:

```
internal/<fitur>/
├── domain/        struct data & aturan murni
├── repository/    query SQL (pgx)
├── service/       logika bisnis & hak akses
└── delivery/http/ handler HTTP (terima request → panggil service → kirim JSON)
```

Alur satu request: **handler → service → repository → database**. Semua fitur dirakit di `cmd/<nama>/main.go`.
Penjelasan lengkap & daftar fitur: README di masing-masing service dan `docs/adr/0011-struktur-service.md`.
Struktur frontend (types / services / stores / hooks / pages): `frontend/README.md` dan `docs/adr/0012-struktur-frontend.md`.

## Menjalankan & mematikan (pakai cmd)

Tiga program utama (plus AI Brain bila memakai menu Dokumen SOP), masing-masing di **jendela cmd sendiri** dan dibiarkan terbuka. Urutan: **core-api → IoT → AI Brain → Frontend**.
Pengaturan (database, kunci, port) dibaca otomatis dari file `.env` di folder masing-masing layanan.

| Program | Folder | Port | Perintah |
|---|---|---|---|
| core-api | `services\core-api-go` | 3346 | `go run ./cmd/api` |
| IoT | `services\iot-ingestion-go` | 3347 (HTTP), 5550 (TCP alat GPS) | `go run ./cmd/iot` |
| AI Brain | `services\ai-brain-py` | 3348 | `.venv\Scripts\uvicorn app.main:build_app --factory --port 3348` (persiapan sekali: README service) |
| Frontend | `frontend` | 8881 | `npm run dev` |

Contoh: `cd services\core-api-go` lalu `go run ./cmd/api`. Tunggu log `HTTP mendengarkan`. Buka http://localhost:8881 dan login.
Cek hidup: http://localhost:3346/health dan http://localhost:3347/health (harus `"status":"ok"`).
`go run` pertama kali mengompilasi ± 10–30 detik; itu normal.

**Mematikan:** klik jendela cmd lalu **Ctrl + C**. Untuk program tanpa jendela: `netstat -ano | findstr :3346 | findstr LISTENING`
(IoT `3347`, frontend `8881`), lalu `taskkill /PID <nomor> /F`.

**Email laporan terjadwal:** PDF dibuat Chrome (harus terpasang) dari halaman cetak frontend, jadi frontend harus menyala
(`FRONTEND_URL` di `.env` core-api). Isi `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` di
`services\core-api-go\.env` lalu jalankan ulang core-api. Tanpa itu jadwal tercatat "Gagal: SMTP belum dikonfigurasi"; Lihat PDF tetap bisa.

**Masalah umum**

| Pesan | Solusi |
|---|---|
| `DATABASE_URL ... wajib diisi` | `.env` tidak ditemukan → `cd` ke folder layanan yang benar |
| `bind: Only one usage of each socket address` | Port masih dipakai program lain → matikan dulu |
| Login gagal "Tidak dapat terhubung ke server" | core-api belum menyala; jalankan ulang frontend setelahnya |
| Peta live "Terputus" | IoT belum menyala; frontend menyambung ulang sendiri |

## Database & migration

Database `simha` (schema `auth`, `core`, `geo`, `iot`, `ai`). Migration dijalankan manual dengan `migrate`, per layanan:
core `services\core-api-go\migrations` (versi terbaru **13**), iot (7), ai (3). Setelah menarik kode baru, jalankan migration lalu **restart layanan**.
Template Excel tiap menu Record / Upload bisa diunduh dari halamannya.

## Aturan penting

- File `.env` dan folder `keys/` berisi rahasia → **tidak** masuk Git (sudah diatur di `.gitignore`).
- Database lama `fms_lite` hanya boleh **dibaca** (referensi), tidak pernah diubah.
- Uji otomatis memakai project **UJI**; project CHR berisi data asli.
