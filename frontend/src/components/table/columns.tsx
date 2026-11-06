// Definisi kolom bersama untuk tabel, tampilan kartu HP, pencarian, dan ekspor Excel.
// Satu definisi kolom = tampil di tabel + kartu HP + file Excel dengan nilai yang sama.
import { Tag, Typography } from 'antd'

import type { ExcelCell } from '@/types/excel.types'
import type { Column, ColumnOptions } from '@/types/table.types'
import { formatDate, formatDateTime, formatNumber, formatPlainDate, formatTime, text } from '@/utils/format'
import { formatCoordinate, googleMapsUrl } from '@/utils/map'
import { toLocal } from '@/utils/time'
import StatusTag, { ActiveTag } from '@/components/ui/StatusTag'

/** Ambil nilai bertingkat "equipment.equipment_code". */
export const pick = (row: unknown, path: string): unknown =>
  path.split('.').reduce<unknown>((o, k) => (o == null ? undefined : (o as Record<string, unknown>)[k]), row)

export const valueOf = <T,>(c: Column<T>, row: T): ExcelCell => (c.value ? c.value(row) : (pick(row, c.key) as ExcelCell))

/** Pembuat kolom siap pakai. */
export const col = {
  text: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({ key, title, sortable: true, ...o }),

  /** Teks tebal (kode unit, kode master). */
  code: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    sortable: true,
    render: (r) => <Typography.Text strong>{text(pick(r, key))}</Typography.Text>,
    ...o,
  }),

  number: <T,>(key: string, title: string, digits = 0, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    align: 'right',
    sortable: true,
    numFmt: digits ? `#,##0.${'0'.repeat(digits)}` : '#,##0',
    render: (r) => formatNumber(pick(r, key) as number, digits),
    ...o,
  }),

  /** Waktu dari server (UTC) → tampil WIB; di Excel sebagai tanggal-waktu lokal. */
  dateTime: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    sortable: true,
    numFmt: 'dd-mm-yyyy hh:mm:ss',
    render: (r) => formatDateTime(pick(r, key) as string),
    value: (r) => excelDate(pick(r, key) as string),
    ...o,
  }),
  date: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    sortable: true,
    numFmt: 'dd-mm-yyyy',
    render: (r) => formatDate(pick(r, key) as string),
    value: (r) => excelDate(pick(r, key) as string),
    ...o,
  }),
  time: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    render: (r) => formatTime(pick(r, key) as string),
    value: (r) => formatTime(pick(r, key) as string),
    ...o,
  }),
  /** Jam saja "HH:mm" (tanpa detik), mis. jam mulai/selesai breakdown, jam timbang. */
  clock: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    render: (r) => formatTime(pick(r, key) as string, false),
    value: (r) => formatTime(pick(r, key) as string, false),
    ...o,
  }),
  /** Tanggal kalender tanpa jam ("2026-10-26"), mis. tanggal operasional. */
  plainDate: <T,>(key: string, title: string, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    sortable: true,
    render: (r) => formatPlainDate(pick(r, key) as string),
    value: (r) => formatPlainDate(pick(r, key) as string),
    ...o,
  }),

  /** Tag berwarna (status/kategori) dengan peta warna & label dari lib/colors. */
  tag: <T,>(key: string, title: string, colors: Record<string, string>, labels?: Record<string, string>, o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    sortable: true,
    render: (r) => <StatusTag value={pick(r, key) as string} colors={colors} labels={labels} />,
    value: (r) => {
      const v = pick(r, key) as string | undefined
      return v ? (labels?.[v.toLowerCase()] ?? v) : null
    },
    ...o,
  }),

  active: <T,>(key = 'is_active', title = 'Status', o: ColumnOptions<T> = {}): Column<T> => ({
    key,
    title,
    render: (r) => <ActiveTag active={!!pick(r, key)} />,
    value: (r) => (pick(r, key) ? 'Active' : 'Inactive'),
    ...o,
  }),

  /** Koordinat dengan link Google Maps. */
  coordinate: <T,>(latKey = 'latitude', lonKey = 'longitude', title = 'Coordinate', o: ColumnOptions<T> = {}): Column<T> => ({
    key: `${latKey},${lonKey}`,
    title,
    render: (r) => {
      const lat = pick(r, latKey) as number | null
      const lon = pick(r, lonKey) as number | null
      const url = googleMapsUrl(lat, lon)
      return url ? <a href={url} target="_blank" rel="noreferrer">{formatCoordinate(lat, lon)}</a> : '-'
    },
    value: (r) => formatCoordinate(pick(r, latKey) as number, pick(r, lonKey) as number),
    ...o,
  }),

  /** Status mesin ON/OFF dari kolom is_engine_on. */
  engine: <T extends { is_engine_on: boolean | null }>(): Column<T> => ({
    key: 'is_engine_on',
    title: 'Engine',
    render: (r) => (r.is_engine_on == null ? '-' : <Tag color={r.is_engine_on ? 'green' : 'default'}>{r.is_engine_on ? 'ON' : 'OFF'}</Tag>),
    value: (r) => (r.is_engine_on == null ? null : r.is_engine_on ? 'ON' : 'OFF'),
  }),

  /** Nomor urut. */
  rowNo: <T,>(): Column<T> => ({ key: '__no', title: 'No', width: 56, align: 'center', mobile: 'hide', render: (_r, i) => i + 1, exportable: false }),
}

/** Waktu ISO → Date dengan jam dinding WIB (Excel tidak mengenal zona waktu). */
function excelDate(iso?: string | null): Date | null {
  if (!iso) return null
  const t = toLocal(iso)
  return new Date(Date.UTC(t.year(), t.month(), t.date(), t.hour(), t.minute(), t.second()))
}
