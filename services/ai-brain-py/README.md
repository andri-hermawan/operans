# ai-brain-py

Layanan AI Brain SIMHA (Python + FastAPI, port **3348**): pustaka SOP, pencarian, dan kelak tanya-jawab bersitasi.
Rancangan: [ADR 0013](../../docs/adr/0013-ai-brain-struktur.md). Status: kerangka, schema `ai`, serta pustaka dokumen (unggah, daftar, baca per halaman) sudah jalan; tanpa LLM/vektor.

## Menjalankan

Sekali saja (butuh Python 3.12):

```bash
py -3.12 -m venv .venv
.venv\Scripts\pip install -e ".[dev]"
```

Siapkan `.env` (salin dari `.env.example`) dan `keys\core_public.pem` (salinan kunci publik core-api), lalu:

```bash
.venv\Scripts\uvicorn app.main:build_app --factory --port 3348
```

Cek: http://localhost:3348/health (200 bila database terhubung; 503 bila tidak). Dokumentasi API otomatis: http://localhost:3348/docs.
Tes: `.venv\Scripts\pytest`.

## Menyiapkan database (sekali)

1. **Admin/DBA** menjalankan `db/admin/001_roles_schema_ai.sql` sebagai `postgres`, lalu memasang password `ai_migrator` & `ai_app`.
2. Migration (alat `migrate` yang sama dengan core-api, pakai akun `ai_migrator`; `search_path=ai,public` wajib agar tabel catatan migrasi dibuat di schema `ai`, bukan `public`):

```bash
migrate -path migrations -database "postgres://ai_migrator:<PASSWORD>@192.168.2.47:5432/simha?sslmode=disable&search_path=ai,public" up
```

## OCR untuk PDF hasil scan (opsional)

Tanpa OCR, PDF yang seluruhnya gambar ditolak. Untuk mengaktifkan (CPU saja, tanpa program tambahan):

```bash
.venv\Scripts\pip install -e ".[ocr]"
```

lalu isi `OCR_ENABLED=true` di `.env` dan jalankan ulang layanan. Model huruf Latin diunduh otomatis saat OCR pertama kali dipakai
(butuh internet sekali; di server tanpa internet, jalankan sekali di mesin ber-internet lalu salin folder `rapidocr\models` dari site-packages).
OCR berjalan di latar belakang: dokumen berstatus **Diproses** (daftar menyegarkan diri), kira-kira beberapa detik per halaman di CPU.
OCR dipakai untuk halaman tanpa teks DAN lampiran bergambar yang hanya memuat kepala/kaki berulang. Dokumen yang diunggah **sebelum** OCR aktif tidak otomatis dibaca ulang: klik ↻ **Proses ulang** pada dokumen itu (admin) setelah OCR aktif. Dokumen yang gagal atau tertahan juga diproses ulang dengan tombol yang sama.

## Endpoint (semua butuh login kecuali /health)

| Endpoint | Siapa |
|---|---|
| `GET /api/v1/departments` · `POST` | semua role melihat · admin menambah |
| `POST /api/v1/documents` (multipart: file PDF, department_id, title, nomor, revisi, roles) | admin |
| `GET /api/v1/documents?department_id=` · `GET /api/v1/documents/{id}` | sesuai hak akses |
| `GET /api/v1/documents/{id}/pages/{n}` (teks halaman; dicatat di audit) | sesuai hak akses |
| `GET /api/v1/search?q=&department_id=&limit=` (hasil: dokumen, halaman, snippet; dicatat di audit) | sesuai hak akses |
| `GET /api/v1/documents/{id}/pages/{n}/image` (halaman sebagai gambar PNG bertanda air pembaca; dicatat di audit) | sesuai hak akses |
| `GET /api/v1/documents/{id}/file` (PDF asli untuk dicetak, **bertanda air** pencetak + waktu; dicatat di audit) | **superadmin saja** |
| `POST /api/v1/documents/{id}/reprocess` (proses ulang dokumen apa pun: baca ulang teks, termasuk OCR) | admin |
| `PUT /api/v1/documents/{id}/roles` · `DELETE /api/v1/documents/{id}` (tarik) | admin |

Pencarian: kata dasar dicocokkan ("keselamatan" menemukan "selamat"); `"frasa persis"`, `kata1 or kata2`, `-kecualikan` didukung. `snippet` adalah teks biasa dengan kata cocok diapit `[[ ]]` — tampilan wajib meng-escape HTML dulu, baru menyorot penanda itu.

Hak akses dokumen = **project + role**: admin/superadmin melihat semua dokumen project aktifnya; dispatcher/viewer hanya dokumen
`ready` yang role-nya didaftarkan pada dokumen itu. Dokumen yang tidak boleh dilihat dijawab **404**. Detail: ADR 0013.

## Peta folder

```
ai-brain-py/
├── app/
│   ├── main.py            ← perakit: konfigurasi, database, pemeriksa token, daftar fitur (tanpa logika bisnis)
│   ├── shared/            ← bagian teknis bersama
│   │   ├── config.py        konfigurasi dari .env
│   │   ├── database.py      kumpulan koneksi PostgreSQL
│   │   ├── security.py      verifikasi token RS256 → Principal (siapa pemanggil)
│   │   └── observability.py log JSON, X-Request-ID, bentuk galat {"error": ...}
│   ├── health/            ← fitur: GET /health
│   ├── me/                ← fitur: GET /api/v1/me (siapa saya menurut token)
│   ├── departments/       ← fitur: pengelompokan dokumen per department (domain · repository · service · router)
│   ├── documents/         ← fitur: unggah PDF, daftar, baca per halaman (+ pdf.py = ekstraksi teks)
│   ├── search/            ← fitur: pencarian kata kunci (full-text bahasa Indonesia)
│   └── audit/             ← catatan audit (dipakai fitur lain)
├── migrations/            ← struktur schema `ai` (golang-migrate) — JANGAN dinomori ulang
├── db/admin/              ← persiapan role & schema oleh admin (sekali)
└── tests/                 ← tes otomatis (kunci RSA & database palsu, tanpa koneksi sungguhan)
```

## Membuat fitur baru (contoh: `documents`)

Salin pola `health`. Alur satu permintaan: **router → service → repository → database**.

1. `app/documents/domain.py` — bentuk data (Pydantic).
2. `app/documents/repository.py` — query SQL (psycopg, selalu berparameter).
3. `app/documents/service.py` — logika bisnis & hak akses (pakai `Principal`).
4. `app/documents/router.py` — endpoint; wajibkan login dengan `Depends(current_principal)`.
5. Daftarkan di `app/main.py`: `app.include_router(documents.router)`.
6. Tambah tes di `tests/`.

Aturan: telemetry tidak masuk AI Brain; angka final selalu dari query (bukan karangan model); tidak ada SQL buatan model.
