// Rumus laporan harian — sama dengan FMS lama (weighbridge.service & ReportPaDailyPage), satu tempat & dites.
import type { CSSProperties } from 'react'

import type { DailyCell } from '@/types/report.types'

import { reportScale } from './colors'
import { dayjs } from './time'

/** Semua tanggal dari `from` sampai `to` (inklusif). */
export function daysBetween(from: string, to: string): string[] {
  const out: string[] = []
  for (let d = dayjs(from); !d.isAfter(dayjs(to), 'day'); d = d.add(1, 'day')) out.push(d.format('YYYY-MM-DD'))
  return out
}

/** Kelompokkan sel per unit: kode → tanggal → nilai. */
export function byUnit(cells: DailyCell[], value: (c: DailyCell) => number): Map<string, Record<string, number>> {
  const m = new Map<string, Record<string, number>>()
  for (const c of cells) {
    const row = m.get(c.equipment_code) ?? {}
    row[c.date] = (row[c.date] ?? 0) + value(c)
    m.set(c.equipment_code, row)
  }
  return new Map([...m.entries()].sort(([a], [b]) => a.localeCompare(b)))
}

const round = (v: number, d = 2) => Math.round(v * 10 ** d) / 10 ** d
const sum = (vs: number[]) => vs.reduce((s, v) => s + v, 0)

/** Ritase = jumlah tiket. */
export const ritase = (c: DailyCell) => c.count
/** Payload (ton/rit) = rata-rata net per tiket ÷ 1000. */
export const payload = (c: DailyCell) => (c.count ? round(c.sum_kg / c.count / 1000) : 0)
/** Produksi (ton) = total net ÷ 1000. */
export const production = (c: DailyCell) => round(c.sum_kg / 1000)
/** Breakdown (jam). */
export const breakdownHours = (c: DailyCell) => round(c.sum_hours)

export const totals = {
  sum: (vs: number[]) => round(sum(vs)),
  /** Rata-rata nilai > 0 (payload). */
  avgPositive: (vs: number[]) => {
    const p = vs.filter((v) => v > 0)
    return p.length ? round(sum(p) / p.length) : 0
  },
  /** Rata-rata semua hari (PA). */
  avg: (vs: number[]) => (vs.length ? round(sum(vs) / vs.length) : 0),
}

/** Jam tersedia per hari: 12 bila satu shift, 24 bila semua shift (sistem lama). */
export const hoursAvailable = (shift?: string) => (shift ? 12 : 24)

/** PA % = (H − jam breakdown) ÷ H × 100; hari tanpa breakdown = 100. */
export const pa = (breakdown: number | undefined, h: number) => round(((h - (breakdown ?? 0)) / h) * 100)

// ---------- warna sel (skala Excel lama) ----------
export const ritaseStyle = (v: number | null): CSSProperties | undefined =>
  !v ? undefined : v < 2 ? reportScale.bad : v <= 3 ? reportScale.warn : reportScale.good
export const productionStyle = (v: number | null): CSSProperties | undefined => (v && v > 0 ? reportScale.good : undefined)
export const breakdownStyle = (limit: number) => (v: number | null): CSSProperties | undefined => (v && v > limit ? reportScale.bad : undefined)
export const paStyle = (v: number | null): CSSProperties | undefined => {
  if (v == null) return undefined
  if (v < 0) return { background: '#111827', color: '#fff' }
  if (v >= 90) return reportScale.good
  if (v >= 75) return { background: '#dbeafe', color: '#1e40af' }
  if (v >= 60) return reportScale.warn
  return reportScale.bad
}
