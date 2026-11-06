// Waktu & shift. SEMUA perhitungan tanggal operasional / shift ada di file ini (ADR 0004):
// data milik shift yang melewati tengah malam dan jamnya sebelum end_time → tanggal operasional = kemarin.
import dayjs, { type Dayjs } from 'dayjs'
import 'dayjs/locale/id'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import timezone from 'dayjs/plugin/timezone'
import utc from 'dayjs/plugin/utc'

import type { OperationalSlot, Shift } from '@/types/shift.types'

dayjs.extend(utc)
dayjs.extend(timezone)
dayjs.extend(customParseFormat)
dayjs.locale('id')

/** Zona waktu project (semua project saat ini WIB). */
export const PROJECT_TZ = 'Asia/Jakarta'

/** Waktu dari server (UTC, ISO) → dayjs di zona project. */
export const toLocal = (iso: string | Date | Dayjs, tz = PROJECT_TZ): Dayjs => dayjs(iso).tz(tz)

/** Jam lokal project "HH:mm" dari waktu ISO (kosong bila tidak ada). */
export const hhmm = (iso: string | null) => (iso ? toLocal(iso).format('HH:mm') : undefined)

/** Menit sejak 00:00 dari "HH:MM". */
export const minutesOf = (hhmm: string): number => {
  const [h, m] = hhmm.split(':').map(Number)
  return h * 60 + (m || 0)
}

/** true bila shift melewati tengah malam (mis. 19:00–07:00). */
export const crossesMidnight = (s: Pick<Shift, 'start_time' | 'end_time'>): boolean =>
  minutesOf(s.end_time) <= minutesOf(s.start_time)

/** Shift yang memuat jam lokal tertentu (menit sejak 00:00). */
export const shiftAt = (shifts: Shift[], minute: number): Shift | undefined =>
  shifts.find((s) => {
    const start = minutesOf(s.start_time)
    const end = minutesOf(s.end_time)
    return crossesMidnight(s) ? minute >= start || minute < end : minute >= start && minute < end
  })

/** Tanggal operasional + shift untuk suatu waktu (default: sekarang). */
export function operationalSlot(shifts: Shift[], at: string | Date | Dayjs = dayjs(), tz = PROJECT_TZ): OperationalSlot {
  const t = toLocal(at, tz)
  const minute = t.hour() * 60 + t.minute()
  const shift = shiftAt(shifts, minute)
  const before = shift && crossesMidnight(shift) && minute < minutesOf(shift.end_time)
  return { date: (before ? t.subtract(1, 'day') : t).format('YYYY-MM-DD'), shift }
}

/**
 * Rentang waktu sebuah shift pada tanggal operasional → [mulai, selesai) sebagai ISO UTC,
 * untuk query ke backend. Tanpa shift = satu hari operasional penuh (mulai shift pertama).
 */
export function shiftRange(date: string, shift: Shift | undefined, shifts: Shift[], tz = PROJECT_TZ): [string, string] {
  const sorted = [...shifts].sort((a, b) => a.sequence - b.sequence)
  const first = shift ?? sorted[0]
  const startHHMM = first?.start_time ?? '00:00'
  const start = dayjs.tz(`${date} ${startHHMM}`, 'YYYY-MM-DD HH:mm', tz)
  let end: Dayjs
  if (shift) {
    end = dayjs.tz(`${date} ${shift.end_time}`, 'YYYY-MM-DD HH:mm', tz)
    if (crossesMidnight(shift)) end = end.add(1, 'day')
  } else {
    end = start.add(1, 'day')
  }
  return [start.toISOString(), end.toISOString()]
}

/** Jam-jam (HH) dalam shift berurutan, melewati tengah malam bila perlu: 19,20,…,23,0,…,6. */
export function shiftHours(shift: Pick<Shift, 'start_time' | 'end_time'>): number[] {
  const start = Math.floor(minutesOf(shift.start_time) / 60)
  const end = Math.ceil(minutesOf(shift.end_time) / 60)
  const count = ((end - start + 24) % 24) || 24
  return Array.from({ length: count }, (_, i) => (start + i) % 24)
}

export { dayjs }
