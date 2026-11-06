import { describe, expect, it } from 'vitest'

import { readableText, speedColor } from './colors'
import { formatDateTime, formatDuration, formatNumber, formatPlainDate, formatUnit } from './format'
import { googleMapsUrl } from './map'

describe('format', () => {
  it('angka id-ID & kosong', () => {
    expect(formatNumber(12345.678, 2)).toBe('12.345,68')
    expect(formatNumber(null)).toBe('-')
    expect(formatUnit(45, 'km/h')).toBe('45 km/h')
  })
  it('waktu server UTC tampil WIB', () => {
    expect(formatDateTime('2026-10-26T00:05:00Z')).toBe('26/10/2026 07:05:00')
    expect(formatPlainDate('2026-10-26')).toBe('26/10/2026')
  })
  it('durasi', () => {
    expect(formatDuration(65)).toBe('1 jam 5 menit')
    expect(formatDuration(0.5)).toBe('30 detik')
  })
})

describe('warna & peta', () => {
  it('pita speed sama dengan sistem lama', () => {
    expect(speedColor(10)).toBe('#111827')
    expect(speedColor(57)).toBe('#dc2626')
  })
  it('teks terbaca', () => {
    expect(readableText('#ffffff')).toBe('#111827')
    expect(readableText('#0b4f8c')).toBe('#ffffff')
  })
  it('link google maps', () => {
    expect(googleMapsUrl(-3.47, 103.87)).toBe('https://www.google.com/maps?q=-3.47,103.87')
    expect(googleMapsUrl(null, 1)).toBeUndefined()
  })
})
