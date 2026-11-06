// Konfigurasi halaman CRUD generik (komponennya: components/crud/CrudPage.tsx).
import type { ReactNode } from 'react'

import type { Resource } from '@/types/auth.types'

import type { Role } from './auth.types'
import type { FilterField, FilterValues } from './filter.types'
import type { FieldDef } from './form.types'
import type { ImportType } from './import.types'
import type { Shift } from './shift.types'
import type { Column } from './table.types'

/** Konteks yang diberikan ke fungsi API konfigurasi. */
export interface CrudCtx {
  projectId: string
  shifts: Shift[]
  role?: Role
  /** shift_code → id (filter daily memakai shift_id). */
  shiftId: (code?: string) => string | undefined
}

export type Payload = Record<string, unknown>

export interface CrudConfig<T> {
  title: string
  resource: Resource
  rowKey: keyof T | ((row: T) => string)
  columns: Column<T>[]
  /** Kunci cache unik halaman. */
  queryKey: string
  list: (filters: FilterValues, ctx: CrudCtx) => Promise<T[]>
  filters?: FilterField[]
  /** Nilai filter awal (dapat memakai tanggal operasional & shift saat ini). */
  defaultFilters?: (now: { date: string; shift?: string }) => FilterValues
  form?: {
    fields: FieldDef[]
    width?: number
    /** Data baris → nilai form (default: baris apa adanya). */
    toForm?: (row: T) => Payload
    /** Nilai awal tambah (selain FieldDef.initial). */
    initial?: (ctx: CrudCtx, filters: FilterValues) => Payload
    create?: (payload: Payload, ctx: CrudCtx) => Promise<unknown>
    update?: (row: T, payload: Payload, ctx: CrudCtx) => Promise<unknown>
    canEdit?: (row: T, ctx: CrudCtx) => boolean
  }
  remove?: { action: (row: T, ctx: CrudCtx) => Promise<unknown>; label: (row: T) => string }
  importType?: ImportType
  exportName?: string
  searchable?: boolean
  /** Aksi tambahan per baris (mis. Reset Password). */
  rowActions?: (row: T, ctx: CrudCtx) => ReactNode
  /** Ringkasan di atas tabel (mis. total net). */
  summary?: (rows: T[]) => ReactNode
  /** Catatan di bawah judul. */
  subtitle?: ReactNode
  /** Batasi isi daftar sesuai role (mis. selain superadmin hanya melihat dirinya). */
  createRoles?: Role[]
}
