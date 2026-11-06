// Hitungan tabel OPERATION PERFORMANCE (Executive Summary): periode & penjumlahan baris.
import type { Cell, Line, Period } from '@/types/report.types'

export const PERIODS: readonly { key: Period; label: string }[] = [
  { key: 'daily', label: 'DAILY' },
  { key: 'mtd', label: 'MTD' },
  { key: 'ytd', label: 'YTD' },
]

/** Jumlah sel per periode dari beberapa baris (baris yang tidak punya periode itu dilewati). */
export const sumCells = (lines: Line[], k: Period): Cell | undefined => {
  const cs = lines.map((l) => l.cells[k]).filter((c): c is Cell => !!c)
  return cs.length ? { plan: cs.reduce((s, c) => s + c.plan, 0), actual: cs.reduce((s, c) => s + c.actual, 0) } : undefined
}
