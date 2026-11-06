import { describe, expect, it } from 'vitest'

import type { Shift } from '@/types/shift.types'

import { operationalSlot, shiftHours, shiftRange } from './time'

const shifts: Shift[] = [
  { id: '1', project_id: 'p', shift_code: 'SHIFT_1', shift_name: 'Shift 1', start_time: '07:00', end_time: '19:00', sequence: 1 },
  { id: '2', project_id: 'p', shift_code: 'SHIFT_2', shift_name: 'Shift 2', start_time: '19:00', end_time: '07:00', sequence: 2 },
]

describe('operationalSlot (ADR 0004)', () => {
  it('siang = shift 1, tanggal hari itu', () => {
    const s = operationalSlot(shifts, '2026-10-23T03:00:00Z') // 10:00 WIB
    expect(s).toMatchObject({ date: '2026-10-23', shift: { shift_code: 'SHIFT_1' } })
  })
  it('lewat tengah malam = shift 2, tanggal kemarin', () => {
    const s = operationalSlot(shifts, '2026-10-22T19:00:00Z') // 23 Okt 02:00 WIB
    expect(s).toMatchObject({ date: '2026-10-22', shift: { shift_code: 'SHIFT_2' } })
  })
  it('19:30 WIB = shift 2, tanggal hari itu', () => {
    expect(operationalSlot(shifts, '2026-10-23T12:30:00Z').date).toBe('2026-10-23')
  })
})

// Halaman pemantauan live memeriksa slot tiap 30 detik; ini titik pergantian yang harus tertangkap.
describe('operationalSlot di titik pergantian (monitoring 24 jam)', () => {
  const at = (wib: string) => {
    const s = operationalSlot(shifts, new Date(`2026-10-27T${wib}+07:00`))
    return `${s.date} ${s.shift?.shift_code}`
  }
  it('18:59 shift 1 → 19:00 shift 2 (tanggal sama)', () => {
    expect(at('18:59:30')).toBe('2026-10-27 SHIFT_1')
    expect(at('19:00:30')).toBe('2026-10-27 SHIFT_2')
  })
  it('06:59 shift 2 (tanggal kemarin-mulai) → 07:00 shift 1 TANGGAL BARU', () => {
    expect(at('06:59:30')).toBe('2026-10-26 SHIFT_2')
    expect(at('07:00:30')).toBe('2026-10-27 SHIFT_1')
  })
})

describe('shiftRange', () => {
  it('shift 2 berakhir besok pagi', () => {
    expect(shiftRange('2026-10-22', shifts[1], shifts)).toEqual(['2026-10-22T12:00:00.000Z', '2026-10-23T00:00:00.000Z'])
  })
  it('tanpa shift = satu hari operasional mulai shift pertama', () => {
    expect(shiftRange('2026-10-22', undefined, shifts)).toEqual(['2026-10-22T00:00:00.000Z', '2026-10-23T00:00:00.000Z'])
  })
})

describe('shiftHours', () => {
  it('melewati tengah malam', () => {
    expect(shiftHours(shifts[1])).toEqual([19, 20, 21, 22, 23, 0, 1, 2, 3, 4, 5, 6])
  })
})
