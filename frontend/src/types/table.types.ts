// Kolom & properti tabel standar (komponennya ada di components/table/).
import type { ReactNode } from 'react'

import type { ExcelCell } from './excel.types'

export interface Column<T> {
  key: string
  title: string
  /** Isi sel di layar (default: teks dari value). */
  render?: (row: T, index: number) => ReactNode
  /** Nilai mentah untuk urut, cari & Excel (default: row[key], boleh "a.b"). */
  value?: (row: T) => ExcelCell
  width?: number
  align?: 'left' | 'center' | 'right'
  fixed?: 'left' | 'right'
  sortable?: boolean
  /** Format Excel, mis. '#,##0.00', 'dd-mm-yyyy hh:mm:ss'. */
  numFmt?: string
  /** 'title' = judul kartu di HP, 'hide' = tidak tampil di HP. */
  mobile?: 'title' | 'hide'
  /** false = tidak ikut diekspor. */
  exportable?: boolean
}

export interface DataTableProps<T> {
  rows: T[] | undefined
  columns: Column<T>[]
  rowKey: keyof T | ((row: T) => string)
  loading?: boolean
  error?: unknown
  onRetry?: () => void
  /** Pencarian teks di semua kolom (di browser). */
  searchable?: boolean
  /** Nama file Excel → tombol "Export Excel" muncul (ekspor SEMUA baris hasil filter, bukan satu halaman). */
  exportName?: string
  /** Tombol/isi tambahan di kiri toolbar. */
  toolbar?: ReactNode
  /** Kolom aksi per baris (Edit/Delete/...). */
  actions?: (row: T) => ReactNode
  pageSize?: number
  onRowClick?: (row: T) => void
  /** Tinggi area scroll tabel (desktop). */
  scrollY?: number | string
}

/** Pengaturan tambahan kolom (menimpa bawaan pembuat kolom). */
export type ColumnOptions<T> = Partial<Column<T>>
