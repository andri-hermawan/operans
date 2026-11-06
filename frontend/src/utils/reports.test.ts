import { describe, expect, it } from 'vitest'

import type { DailyCell } from '@/types/report.types'

import { byUnit, daysBetween, pa, payload, production, ritase, totals } from './reports'

const cells: DailyCell[] = [
  { equipment_code: 'DT02', date: '2026-10-25', count: 3, sum_kg: 54000, sum_hours: 0 },
  { equipment_code: 'DT01', date: '2026-10-25', count: 2, sum_kg: 36500, sum_hours: 0 },
  { equipment_code: 'DT01', date: '2026-10-26', count: 4, sum_kg: 70000, sum_hours: 0 },
]

describe('laporan harian (rumus FMS lama)', () => {
  it('daftar tanggal inklusif', () => {
    expect(daysBetween('2026-10-30', '2026-11-02')).toEqual(['2026-10-30', '2026-10-31', '2026-11-01', '2026-11-02'])
  })
  it('ritase, payload, produksi per unit (urut kode)', () => {
    const r = byUnit(cells, ritase)
    expect([...r.keys()]).toEqual(['DT01', 'DT02'])
    expect(r.get('DT01')).toEqual({ '2026-10-25': 2, '2026-10-26': 4 })
    expect(byUnit(cells, payload).get('DT01')!['2026-10-25']).toBe(18.25)
    expect(byUnit(cells, production).get('DT01')!['2026-10-26']).toBe(70)
  })
  it('total: payload = rata-rata nilai > 0', () => {
    expect(totals.avgPositive([18.25, 0, 17.5])).toBe(17.88)
    expect(totals.sum([1.111, 2.222])).toBe(3.33)
  })
  it('PA = (H − breakdown) / H × 100', () => {
    expect(pa(3, 12)).toBe(75)
    expect(pa(undefined, 24)).toBe(100)
  })
})
