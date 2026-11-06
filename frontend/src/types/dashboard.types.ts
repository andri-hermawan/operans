// Tipe Dashboard: ringkasan operasi per tanggal operasional + shift.
import type { ShiftPerformanceRow } from './report.types'

/** Baris ringkasan satu unit: kinerja shift + status live + Physical Availability. */
export type DashboardUnitRow = ShiftPerformanceRow & { status: string; pa: number }

export interface DashboardKpi {
  units: number
  online: number
  running: number
  ritase: number
  tonnes: number
  /** ton per rit */
  payload: number
  /** Physical Availability % armada */
  pa: number
  /** Utilization % = jam running ÷ jam tersedia */
  utilization: number
  mileage: number
  avgSpeed: number
  fuelUsed: number
  fuelFilled: number
  burn: number
  alerts: number
  breakdownHours: number
}

export interface TimeUsage {
  running: number
  idle: number
  breakdown: number
  /** sisa jam tersedia (stop / tidak ada data) */
  other: number
}

export interface AlertHourRow {
  hour: string
  counts: Record<string, number>
}
