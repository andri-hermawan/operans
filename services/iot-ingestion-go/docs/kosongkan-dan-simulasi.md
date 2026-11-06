# Panduan: Mengosongkan Data & Menjalankan Simulator DT03

Panduan ini untuk menguji apakah **sistem baru (SIMHA)** menghasilkan data yang **sama** dengan
**sistem lama (FMS)**, memakai simulator yang sama.

---

## Gambaran singkat

```
Sistem LAMA                                   Sistem BARU
-----------                                   -----------
simulator-api-DT03.js  (Node.js)              simulator-dt03  (Go)
   │  150 data, satu per satu tiap 3 detik       │  150 data yang SAMA, satu per satu tiap 3 detik
   ▼                                             ▼
API FMS lama                                  Layanan IoT SIMHA (port 8081)
   ▼                                             ▼
database fms_lite                             database simha (schema iot)
```

- Kedua simulator mengirim **data yang sama** (posisi, kecepatan, BBM, mesin hidup/mati) dengan **jam data yang sama**
  (mulai 26-10-2026 07:05:00, tiap data berjarak 30 detik).
- Data **tidak disalin** dari fms_lite. Simulator baru mengirim data ke sistem baru, lalu sistem baru
  mengolahnya sendiri (zona, status, BBM, alert, …).
- Setelah selesai, hasil di kedua database **dibandingkan**. fms_lite hanya **dibaca**, tidak pernah diubah.

---

## Yang perlu ada dulu

| Kebutuhan | Cara cek |
|---|---|
| Go terpasang | buka PowerShell, ketik `go version` |
| psql terpasang | ketik `psql --version` |
| File `.env` dan `.env.migrate` di folder `services\iot-ingestion-go` | sudah ada (berisi alamat database, jangan dibagikan) |
| File `keys\simulator-dt03.key` | berisi API key alat `463719065679574` (jangan dibagikan) |
| Data simulator lama sudah ada di fms_lite | simulator lama sudah pernah dijalankan untuk 26-10-2026 |

---

## Langkah demi langkah


### Langkah 1: Matikan layanan IoT

Kalau layanan IoT dijalankan di terminal, tekan **Ctrl + C** di terminal itu.
Kalau berjalan di belakang layar:

```bash
Stop-Process -Name iot -Force
```

### Langkah 2: Kosongkan tabel IoT

Tabel yang dikosongkan:

| Tabel | Isinya |
|---|---|
| `iot.telemetry_events` | semua data yang dikirim alat/simulator |
| `iot.equipment_live_states` | kondisi terakhir tiap unit |
| `iot.geofence_events` | kejadian masuk/keluar area |
| `iot.alerts` | alert (overspeed, underspeed, off track, BBM turun) |
| `iot.fuel_events` | kejadian isi/turun BBM |
| `iot.notifications` | antrian pesan WhatsApp (ikut dikosongkan karena terhubung ke `iot.alerts`) |

Yang **tidak** dikosongkan: daftar alat (`iot.devices`), data unit/peta/aturan (schema `core` dan `geo`), dan fms_lite.

Query PostgreSQL (jalankan di database **simha**, user **iot_migrator**, misalnya lewat pgAdmin / DBeaver):

```sql
BEGIN;
TRUNCATE iot.notifications,
         iot.fuel_events,
         iot.alerts,
         iot.geofence_events,
         iot.equipment_live_states,
         iot.telemetry_events
  RESTART IDENTITY;
COMMIT;
```

- `TRUNCATE` = hapus semua isi tabel (jauh lebih cepat dari `DELETE`).
- Semua tabel ditulis dalam **satu** perintah supaya tidak ditolak karena tabel saling terhubung.
- `RESTART IDENTITY` = nomor id mulai lagi dari 1.

Cek hasilnya (semua harus 0):

```sql
SELECT 'telemetry_events' AS tabel, count(*) FROM iot.telemetry_events
UNION ALL SELECT 'equipment_live_states', count(*) FROM iot.equipment_live_states
UNION ALL SELECT 'geofence_events', count(*) FROM iot.geofence_events
UNION ALL SELECT 'alerts', count(*) FROM iot.alerts
UNION ALL SELECT 'fuel_events', count(*) FROM iot.fuel_events
UNION ALL SELECT 'notifications', count(*) FROM iot.notifications;
```


<details>
<summary>Hanya ingin menghapus data SATU unit? (klik)</summary>

Hapus berurutan dari tabel "anak" ke "induk":

```sql
BEGIN;
DELETE FROM iot.notifications
 WHERE alert_id IN (SELECT id FROM iot.alerts WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3');
DELETE FROM iot.fuel_events           WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3';
DELETE FROM iot.alerts                WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3';
DELETE FROM iot.geofence_events       WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3';
DELETE FROM iot.equipment_live_states WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3';
DELETE FROM iot.telemetry_events      WHERE equipment_id = 'f070646a-2ff8-416a-bb9f-f5f940ddb4c3';
COMMIT;
```
</details>

### Langkah 3: Nyalakan layanan IoT

Buka PowerShell (terminal 1) di folder `services\iot-ingestion-go`:

```bash
Get-Content .env | ForEach-Object { if ($_ -match '^\s*([^#=]+)=(.*)$') { [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim()) } }
```

```bash
go run ./cmd/iot
```

Baris pertama memuat isi `.env`, baris kedua menyalakan layanan. Biarkan terminal ini terbuka.

### Langkah 4: Jalankan simulator (`toolssimulator-dt03`)

Buka PowerShell baru (terminal 2) di folder yang sama:

```bash
go run ./tools/simulator-dt03 -key (Get-Content keys\simulator-dt03.key)
```

Di layar akan muncul satu baris untuk setiap data, mirip simulator lama:

```
#001 07:05:00 | Engine=ON  | Speed= 0 | Fuel=1950 | TERSIMPAN
#002 07:05:30 | Engine=ON  | Speed= 0 | Fuel=1949.025 | TERSIMPAN
...
RINGKASAN dikirim=150 tersimpan=150 duplikat=0
```

- **TERSIMPAN** = data diterima sistem baru.
- **DUPLIKAT** = data dengan jam yang sama sudah pernah dikirim (kosongkan tabel dulu kalau mau mengulang).


## Kalau ada masalah

| Pesan | Penyebab & solusi |
|---|---|
| `API key tidak ada` | file `keys\simulator-dt03.key` belum ada → minta API key alat ke admin |
| `layanan IoT tidak menyala` | lihat log di `%TEMP%\simha-iot.err`; biasanya isi `.env` salah atau database tidak bisa dihubungi |
| Simulator: `connection refused` | layanan IoT belum menyala (Langkah 3) |
| Semua data `DUPLIKAT` | tabel belum dikosongkan (Langkah 2) |
