// Isian filter halaman. Nilainya disimpan di URL (lihat hooks/useFilters.ts).
import type { ReactNode } from 'react'

export type FilterValues = Record<string, string | undefined>

/** Jenis isian filter. Pilihan shift & unit SELALU dari data project (bukan "Shift 1/2" tetap). */
export type FilterField =
  | { type: 'date'; key?: string; label?: string; allowClear?: boolean }
  | { type: 'dateRange'; from?: string; to?: string; label?: string; maxDays?: number }
  | { type: 'shift'; key?: string; label?: string; allowAll?: boolean }
  | { type: 'equipment'; key?: string; label?: string; multiple?: boolean }
  | { type: 'search'; key?: string; label?: string; placeholder?: string }
  | { type: 'select'; key: string; label: string; options: { value: string; label: ReactNode }[]; allowAll?: boolean; multiple?: boolean }
