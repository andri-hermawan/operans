// Format tampilan. Satu-satunya tempat untuk format angka, tanggal, durasi & nilai kosong.
import { toLocal } from './time'

export const EMPTY = '-'

const isBlank = (v: unknown): v is null | undefined | '' => v === null || v === undefined || v === ''

const nf = (digits: number) =>
  new Intl.NumberFormat('id-ID', { minimumFractionDigits: digits, maximumFractionDigits: digits })

/** Angka id-ID: 12.345,67. */
export const formatNumber = (v: number | string | null | undefined, digits = 0): string => {
  if (isBlank(v)) return EMPTY
  const n = Number(v)
  return Number.isFinite(n) ? nf(digits).format(n) : EMPTY
}

/** Angka + satuan: "45 km/h", "120,50 L". */
export const formatUnit = (v: number | string | null | undefined, unit: string, digits = 0): string =>
  isBlank(v) ? EMPTY : `${formatNumber(v, digits)} ${unit}`

export const formatPercent = (v: number | string | null | undefined, digits = 0): string =>
  isBlank(v) ? EMPTY : `${formatNumber(v, digits)}%`

/** Tanggal/waktu dari server (UTC) ditampilkan di jam project. */
export const formatDate = (iso: string | null | undefined, pattern = 'DD/MM/YYYY'): string =>
  iso ? toLocal(iso).format(pattern) : EMPTY
export const formatDateTime = (iso: string | null | undefined): string => formatDate(iso, 'DD/MM/YYYY HH:mm:ss')
export const formatTime = (iso: string | null | undefined, withSeconds = true): string =>
  formatDate(iso, withSeconds ? 'HH:mm:ss' : 'HH:mm')

/** Tanggal kalender tanpa jam ("2026-10-26") → "26/10/2026" (tanpa geser zona). */
export const formatPlainDate = (ymd: string | null | undefined): string => {
  if (!ymd) return EMPTY
  const [y, m, d] = ymd.slice(0, 10).split('-')
  return d && m && y ? `${d}/${m}/${y}` : ymd
}

/** Durasi dalam menit → "1 jam 5 menit" / "45 menit" / "30 detik". */
export const formatDuration = (minutes: number | null | undefined): string => {
  if (isBlank(minutes) || !Number.isFinite(minutes)) return EMPTY
  if (minutes < 1) return `${Math.round(minutes * 60)} detik`
  const h = Math.floor(minutes / 60)
  const m = Math.round(minutes % 60)
  return h ? `${h} jam${m ? ` ${m} menit` : ''}` : `${m} menit`
}

/** Selisih menit antara dua waktu ISO (null bila salah satu kosong). */
export const minutesBetween = (from?: string | null, to?: string | null): number | null =>
  from && to ? (new Date(to).getTime() - new Date(from).getTime()) / 60000 : null

/** Teks bebas: kosong → "-". */
export const text = (v: unknown): string => (isBlank(v) ? EMPTY : String(v))

/** Waktu ISO → "baru saja" / "5 menit lalu" / "2 jam lalu" / "3 hari lalu". */
export const formatRelative = (iso: string | null | undefined): string => {
  if (!iso) return EMPTY
  const sec = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (sec < 60) return 'baru saja'
  if (sec < 3600) return `${Math.floor(sec / 60)} menit lalu`
  if (sec < 86400) return `${Math.floor(sec / 3600)} jam lalu`
  return `${Math.floor(sec / 86400)} hari lalu`
}
