import { describe, expect, it } from 'vitest'

import { DAY_KEYS, MONTH_KEYS, MONTH_NAMES, dayTitle, daysInMonth, yearOptions } from './plan'

describe('kolom rencana', () => {
  it('12 bulan dan 31 tanggal, urutan sama dengan backend', () => {
    expect(MONTH_KEYS).toHaveLength(12)
    expect(MONTH_NAMES).toHaveLength(12)
    expect(MONTH_KEYS[0]).toBe('jan')
    expect(MONTH_KEYS[11]).toBe('dec')
    expect(DAY_KEYS).toHaveLength(31)
    expect(DAY_KEYS[0]).toBe('day_01')
    expect(DAY_KEYS[30]).toBe('day_31')
  })

  it('judul kolom tanggal mengikuti tabel lama (T01..T31)', () => {
    expect(dayTitle('day_01')).toBe('T01')
    expect(dayTitle('day_31')).toBe('T31')
  })
})

describe('daysInMonth', () => {
  it('panjang bulan, termasuk Februari kabisat', () => {
    expect(daysInMonth(2025, 6)).toBe(30)
    expect(daysInMonth(2025, 1)).toBe(31)
    expect(daysInMonth(2025, 2)).toBe(28)
    expect(daysInMonth(2024, 2)).toBe(29)
    expect(daysInMonth(2100, 2)).toBe(28) // 2100 bukan kabisat
  })
})

describe('yearOptions', () => {
  it('tahun ini ± 5 secara bawaan, nilai berupa teks', () => {
    const o = yearOptions(2026)
    expect(o).toHaveLength(11)
    expect(o[0]).toEqual({ value: '2021', label: '2021' })
    expect(o[5].value).toBe('2026')
    expect(o[10].value).toBe('2031')
  })
})
