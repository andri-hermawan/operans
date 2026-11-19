import { describe, expect, it } from 'vitest'

import { findHaulCycles, formatSeconds, type HaulPoint } from './haulCycle'

const T0 = Date.UTC(2026, 10, 14, 0, 0, 0)
let t = 0
/** satu bacaan: dt detik setelah bacaan sebelumnya; speed bawaan 40 km/jam (bergerak). */
const pt = (segment: string | null, vessel: string, dt = 3, speed = 40): HaulPoint => {
  t += dt
  return { event_time: new Date(T0 + t * 1000).toISOString(), segment_name: segment, vessel_status: vessel, speed_kph: speed }
}
const many = (n: number, segment: string | null, vessel: string, dt = 3, speed = 40) => Array.from({ length: n }, () => pt(segment, vessel, dt, speed))

describe('findHaulCycles', () => {
  it('siklus lengkap: mulai saat speed 0, berakhir saat keluar zona (tanpa cek status empty)', () => {
    t = 0
    const rows = [
      ...many(5, 'Km 02+000 - 03+000', 'loaded'),
      ...many(4, 'Km 01+000 - 02+000', 'loaded', 3, 20), // masuk, masih bergerak
      ...many(10, 'Km 01+000 - 02+000', 'loaded', 60, 0), // antri 600 dtk (speed 0)
      ...many(10, 'Jembatan Timbang Km 01', 'loaded', 150, 0), // antri di jembatan timbang 1500 dtk
      ...many(3, 'Km 00+000 - 01+000', 'loaded'), // meninggalkan area antri
      ...many(5, 'Stockpile Emplacement', 'loaded', 60, 2),
      pt('Trase Workshop', 'loaded'), // 1 titik di tengah zona bukan "keluar"
      ...many(3, 'Stockpile Emplacement', 'loaded', 60, 2),
      ...many(4, 'Trase Emplacement', 'loaded'),
      ...many(5, 'Km 00+000 - 01+000', 'empty'),
    ]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(1)
    const x = c[0]
    expect(x.ongoing).toBe(false)
    expect(rows[x.startIdx].speed_kph).toBe(0)
    expect(x.startIdx).toBeGreaterThan(x.entryIdx) // mulai antri sesudah masuk
    expect(x.seconds).toBeGreaterThan(2000)
    expect(rows[x.endIdx].segment_name).toBe('Trase Emplacement') // titik pertama di luar zona
  })

  it('antrian berjam-jam (DT10250: ±5 jam) tidak dibatalkan dan mulai dihitung dari speed 0 pertama', () => {
    t = 0
    const rows = [
      ...many(4, 'Km 02+000 - 03+000', 'loaded'),
      ...many(3, 'Km 01+000 - 02+000', 'loaded', 3, 20),
      ...many(1700, 'Km 01+000 - 02+000', 'loaded', 10, 0), // ±4,7 jam diam
      ...many(3, 'Km 00+000 - 01+000', 'loaded'),
      ...many(20, 'Trase Workshop', 'loaded', 60, 0),
      ...many(20, 'Stockpile Emplacement', 'loaded', 60, 1),
      ...many(4, 'Trase Workshop', 'loaded'), // keluar zona, tetap loaded (belum empty)
    ]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(1)
    expect(rows[c[0].startIdx].event_time).toBe(rows[c[0].entryIdx + 3].event_time) // speed 0 pertama, bukan di tengah antrian
    expect(c[0].seconds).toBeGreaterThan(4 * 3600)
  })

  it('unit lewat tanpa berhenti: mulai dihitung dari saat masuk', () => {
    t = 0
    const rows = [...many(4, 'Km 01+000 - 02+000', 'loaded'), ...many(3, 'Km 00+000 - 01+000', 'loaded'), ...many(4, 'Stockpile Emplacement', 'loaded', 60, 5), ...many(4, 'Trase Emplacement', 'empty')]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(1)
    expect(c[0].startIdx).toBe(c[0].entryIdx)
  })

  it('TLS2 Zona 4 juga dianggap zona tujuan', () => {
    t = 0
    const rows = [...many(4, 'Km 01+000 - 02+000', 'loaded'), ...many(6, 'TLS2 Zona 4', 'loaded', 30), ...many(4, 'Trase Emplacement', 'empty')]
    expect(findHaulCycles(rows)).toHaveLength(1)
  })

  it('kedipan loaded 1–2 bacaan pada unit kosong tidak membuat siklus', () => {
    t = 0
    const rows = [
      ...many(5, 'Km 01+000 - 02+000', 'empty'),
      pt('Km 01+000 - 02+000', 'loaded'),
      pt('Km 01+000 - 02+000', 'loaded'),
      ...many(8, 'Km 01+000 - 02+000', 'empty'),
      ...many(5, 'Km 02+000 - 03+000', 'empty'),
    ]
    expect(findHaulCycles(rows)).toHaveLength(0)
  })

  it('masuk palsu (loaded sesaat lalu kosong) dibatalkan, siklus asli sesudahnya tetap terhitung', () => {
    t = 0
    const rows = [
      ...many(3, 'Km 01+000 - 02+000', 'loaded'),
      ...many(8, 'Km 02+000 - 03+000', 'empty'), // kosong ≥ 5 bacaan → batal
      ...many(5, 'Km 02+000 - 03+000', 'empty'),
      ...many(4, 'Km 01+000 - 02+000', 'loaded'),
      ...many(4, 'Stockpile Emplacement', 'loaded', 60),
      ...many(4, 'Trase Emplacement', 'loaded'),
    ]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(1)
    expect(c[0].entryIdx).toBe(16)
  })

  it('data berakhir saat masih di dalam zona → ongoing', () => {
    t = 0
    const rows = [...many(4, 'Km 01+000 - 02+000', 'loaded'), ...many(5, 'Stockpile Emplacement', 'loaded', 60)]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(1)
    expect(c[0].ongoing).toBe(true)
  })

  it('dua ritase dalam satu shift menghasilkan dua siklus', () => {
    t = 0
    const one = [
      ...many(4, 'Km 01+000 - 02+000', 'loaded', 3, 20),
      ...many(8, 'Km 01+000 - 02+000', 'loaded', 30, 0),
      ...many(3, 'Km 00+000 - 01+000', 'loaded'),
      ...many(4, 'Stockpile Emplacement', 'loaded', 60),
      ...many(4, 'Trase Emplacement', 'empty'),
      ...many(6, 'Km 05+000 - 06+000', 'empty'),
    ]
    const rows = [...one, ...one]
    const c = findHaulCycles(rows)
    expect(c).toHaveLength(2)
  })
})

describe('formatSeconds', () => {
  it('format', () => {
    expect(formatSeconds(45)).toBe('45 dtk')
    expect(formatSeconds(2242)).toBe('37 mnt 22 dtk')
    expect(formatSeconds(3900)).toBe('1 j 05 mnt')
  })
})
