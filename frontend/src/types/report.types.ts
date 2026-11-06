// Tipe laporan: rekap harian per unit (core-api) dan tabel matriks laporan.
import type { CSSProperties, ReactNode } from 'react'

import type { Role } from './auth.types'

export interface DailyCell {
  equipment_code: string
  date: string // YYYY-MM-DD
  count: number // weighbridge: ritase
  sum_kg: number // weighbridge: total net kg
  sum_hours: number // breakdown: jam
}

/** Executive Summary Hauling: produksi (net kg) satu customer pada tanggal laporan, bulan berjalan (MTD), dan tahun berjalan (YTD). */
export interface ExecutiveHaulingRow {
  customer: string
  daily_kg: number
  mtd_kg: number
  ytd_kg: number
}

/** Executive Summary Emplacement: per partner, produksi = Total TLS Out (ton) dan inventory = Sisa Cargo (ton) pada tanggal laporan. */
export interface ExecutiveEmplacementRow {
  partner: string
  daily_production: number
  mtd_production: number
  ytd_production: number
  inventory: number
}

export interface ReportPeriod {
  from?: string
  to?: string
  shiftId?: string
  codes?: string[]
}

export interface MatrixRow {
  code: string
  values: Record<string, number | null> // tanggal → nilai (null = tidak ada data)
  total: number | null
}

export interface MatrixProps {
  title: string
  days: string[]
  rows: MatrixRow[]
  uom: string
  /** Tampilan angka sel. */
  format: (v: number | null) => string
  /** Warna sel (skala laporan di utils/colors). */
  cellStyle?: (v: number | null) => CSSProperties | undefined
  totalTitle: string
  loading?: boolean
  exportName: string
  /** Angka desimal di Excel. */
  digits?: number
}

// ---- konfigurasi laporan matriks harian (components/report/DailyMatrix.tsx)
export type DailyMatrixKind = 'ritase' | 'payload' | 'production' | 'breakdown' | 'pa'

export interface DailyMatrixConfig {
  title: string
  source: 'weighbridge' | 'breakdown'
  uom: (shift?: string) => string
  value: (c: DailyCell) => number
  digits: number
  total: { title: string; fn: (vs: number[]) => number }
  style: (shift?: string) => MatrixRowStyle
  /** PA: nilai per hari dihitung dari jam breakdown (hari tanpa data tetap dihitung). */
  derive?: (perDay: Record<string, number>, days: string[], shift?: string) => Record<string, number>
}
type MatrixRowStyle = MatrixProps['cellStyle']

export type DailyReportKind = 'weighbridge-daily' | 'breakdown-daily'

/** Baris laporan Equipment Shift Performance. */
export interface ShiftPerformanceRow {
  code: string
  running: number
  idle: number
  avg: number
  max: number
  mileage: number
  fuelUsed: number
  burn: number
  ritase: number
  production: number
  breakdown: number
  alerts: number
}

/** Satu kartu laporan di halaman Report. */
export interface ReportMenuItem {
  path: string
  title: string
  desc: string
  icon: ReactNode
  roles?: Role[]
}

// Tabel OPERATION PERFORMANCE (Executive Summary)
export type Period = 'daily' | 'mtd' | 'ytd'
export type Cell = { plan: number; actual: number }
export type Line = { label: string; cells: Partial<Record<Period, Cell>>; bold?: boolean }
export type Section = { title: string; lines: Line[] }
