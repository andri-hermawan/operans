# SIMHA Web (frontend)

Aplikasi web SIMHA: React + TypeScript + Vite + Ant Design. Rancangan: ADR 0010
(`docs/adr/0010-frontend.md`). Layout & nama menu mengikuti FMS lama.

## Menjalankan di komputer sendiri

Yang harus sudah menyala: **core-api** (port 3346) dan **layanan IoT** (port 5550).

1. Buka PowerShell di folder ini (`operans\frontend`).
2. Sekali saja (atau setelah `package.json` berubah), pasang paket:

   ```bash
   npm install
   ```

3. Jalankan:

   ```bash
   npm run dev
   ```

4. Buka **http://localhost:8881** lalu login dengan akun SIMHA.

Perintah lain:

| Perintah | Gunanya |
|---|---|
| `npm test` | menjalankan unit test (format, waktu/shift, warna) |
| `npm run lint` | memeriksa gaya & kesalahan umum kode |
| `npm run build` | membuat versi produksi di folder `dist` |

## Bagaimana browser bicara ke backend

Browser hanya bicara ke Vite (8881). Vite meneruskan:

| Alamat di browser | Diteruskan ke |
|---|---|
| `/core/...` | core-api `http://localhost:3346/...` |
| `/iot/...` (termasuk WebSocket `/iot/ws/live`) | layanan IoT `http://localhost:5550/...` |

Alamat lain bisa diatur di `.env` (contoh: `.env.example`). Di produksi Nginx melakukan hal yang sama.

## Peta folder (di mana menaruh apa)

Aturan: **satu jenis berkas = satu folder.** Tidak ada hook di `components/`, tidak ada tipe di `pages/`, tidak ada
`fetch` di luar `services/`. Alasan & peta pemindahan: ADR 0012 (`docs/adr/0012-struktur-frontend.md`).

```
src/
├── main.tsx, App.tsx     ← pintu masuk (mulai membaca dari sini)
├── config/               tema antd (theme.ts), QueryClient (queryClient.ts)
├── types/                SEMUA tipe data: satu berkas per domain (equipment.types.ts, alert.types.ts, ...)
├── services/
│   ├── http/             client.ts = satu-satunya fetch: token, perpanjang sesi otomatis, pesan galat
│   └── api/              SEMUA panggilan backend: satu berkas per domain (shift.api.ts, user.api.ts, ...)
├── stores/               zustand: auth.store.ts (login), ui.store.ts (menu samping)
├── hooks/                SEMUA hook: useShifts, useAlerts, useFilters (filter di URL), useLiveSocket, ...
├── utils/                fungsi biasa tanpa tampilan + tesnya (npm test)
│                         format (angka, tanggal WIB), time (tanggal operasional & shift, ADR 0004),
│                         colors (SEMUA warna), excel, permissions (tabel hak per role), ...
├── router/               routes.tsx ← SATU daftar halaman: menu, judul, ikon, hak akses
│                         index.tsx (router), guards/ (wajib login, cek hak akses)
├── layouts/              MainLayout + Header + Sidebar
├── components/           komponen dipakai banyak halaman
│   ├── ui/ feedback/     PageHeader, SectionCard, StatusTag, FormDrawer, RouteError, ...
│   ├── table/            DataTable + col.* (satu definisi kolom → tabel, kartu HP, Excel)
│   ├── form/ crud/       FormFields, CrudPage (daftar + form + hapus + Excel dari SATU konfigurasi), ImportModal
│   ├── filters/          FilterBar
│   ├── map/ chart/       BaseMap, MapShell, layer peta, Chart (ECharts)
│   └── report/ history/ alert/ monitoring/   komponen generik per kelompok halaman
└── pages/                satu folder per menu, SATU halaman per berkas (default export)
    ├── master/user/UserPage.tsx   (+ components/ untuk bagian khusus halaman itu)
    └── ...
```

Alur membaca dari layar ke server:

```
pages/…Page.tsx → hooks/useXxx.ts → services/api/xxx.api.ts → services/http/client.ts → backend
                                          (bentuk data: types/xxx.types.ts)
```

## Membuat halaman daftar/form baru (contoh)

1. **Tipe** di `types/shift.types.ts`, **panggilan API** di `services/api/shift.api.ts`:

   ```ts
   export const listShiftSettings = async (projectId: string) =>
     (await core.get<ListResponse<ShiftRow>>('/api/v1/shifts/manage', { params: { project_id: projectId } })).data
   export const createShift = (body: Record<string, unknown>) => core.post('/api/v1/shifts', body)
   export const updateShift = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/shifts/${id}`, body)
   ```

2. **Halaman** `pages/master/shift/ShiftPage.tsx` cukup berupa konfigurasi `CrudPage`:

   ```tsx
   const shiftConfig: CrudConfig<ShiftRow> = {
     title: 'Shift', resource: 'master', rowKey: 'id', queryKey: 'shifts-manage',
     list: (_f, ctx) => listShiftSettings(ctx.projectId),
     columns: [col.code('shift_code', 'Code'), col.text('shift_name', 'Name'), col.active()],
     form: {
       fields: [{ name: 'shift_code', label: 'Code', type: 'upper', required: true } /* ... */],
       create: (p, ctx) => createShift({ ...p, project_id: ctx.projectId }),
       update: (r, p) => updateShift(r.id, p),
     },
     exportName: 'shift',          // tombol Export Excel
     // importType: 'weighbridge-tickets'  → menu Upload Excel (template + pratinjau + simpan)
   }
   export default function ShiftPage() {
     return <CrudPage config={shiftConfig} />
   }
   ```

3. **Daftarkan satu baris** di `router/routes.tsx`. Tabel, kartu HP, filter, form, hapus, Excel, izin per role sudah otomatis.

**Aturan:** halaman tidak menulis ulang format, warna, filter, tabel, atau ekspor sendiri — selalu memakai
`utils/` dan `components/`. Kalau butuh sesuatu yang belum ada dan akan dipakai lebih dari satu halaman, tambahkan
ke `utils/` atau `components/`, bukan ke halaman. Data dari server lewat TanStack Query (`hooks/`); zustand hanya untuk
keadaan di sisi aplikasi (login, menu samping).

## Login & keamanan

- Access token hanya di memori; refresh token di `localStorage` (`simha.refresh`) agar tetap login saat halaman
  dimuat ulang. Sesi diperpanjang otomatis sekali saat server menjawab 401.
- Hak akses menu & halaman: `src/utils/permissions.ts` (superadmin, admin, dispatcher, viewer).
- Jangan menulis email/password/token di kode.
