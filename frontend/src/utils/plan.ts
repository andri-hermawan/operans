// Bantuan Plan Yearly / Plan Monthly (fungsi biasa, tanpa tampilan).
import type { DayKey, MonthKey } from '@/types/plan.types'

/** Kolom bulan, urutan sama dengan tabel & template Excel. */
export const MONTH_KEYS: readonly MonthKey[] = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec']

export const MONTH_TITLES: Record<MonthKey, string> = {
  jan: 'Jan', feb: 'Feb', mar: 'Mar', apr: 'Apr', may: 'May', jun: 'Jun', jul: 'Jul', aug: 'Aug', sep: 'Sep', oct: 'Oct', nov: 'Nov', dec: 'Dec',
}

export const MONTH_NAMES = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember']

/** day_01 … day_31. */
export const DAY_KEYS: readonly DayKey[] = Array.from({ length: 31 }, (_, i) => `day_${String(i + 1).padStart(2, '0')}` as DayKey)

/** "day_07" → "T07" (judul kolom seperti tabel/template lama). */
export const dayTitle = (key: DayKey): string => `T${key.slice(4)}`

/** Jumlah hari pada bulan itu (Februari memperhitungkan tahun kabisat). */
export const daysInMonth = (year: number, month: number): number => new Date(Date.UTC(year, month, 0)).getUTCDate()

/** Pilihan tahun untuk filter: tahun ini ± span. */
export const yearOptions = (currentYear: number, span = 5): { value: string; label: string }[] =>
  Array.from({ length: span * 2 + 1 }, (_, i) => String(currentYear - span + i)).map((y) => ({ value: y, label: y }))
