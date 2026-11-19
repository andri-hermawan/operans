// Siklus antrian hauling: unit bermuatan (loaded) masuk Km 01+000 - 02+000, antri (speed 0), lalu keluar dari
// Stockpile Emplacement / TLS2 Zona 4. Antrian terjadi di jalur ini, jadi hanya bagian ini yang dihitung.
//
// Aturan (analisa telemetry_events DT10346 18/11 dan DT10250 14/11, 2026-11-19):
//  - Masuk  : ≥ 3 bacaan berturut-turut di Km 01+000 - 02+000 dengan vessel_status = loaded
//             (sensor muatan sering berkedip loaded/empty saat melaju; satu bacaan tidak cukup).
//  - Mulai  : bacaan pertama dari ≥ 2 bacaan speed 0 berturut-turut di Km 01+000 - 02+000 / Jembatan Timbang Km 01
//             (= mulai antri). Bila unit lewat tanpa berhenti, waktu masuk dipakai.
//  - Akhir  : titik pertama di luar zona tujuan, setelah ≥ 3 bacaan berturut-turut di luar zona (jeda 1 titik
//             "Trase Workshop" di tengah zona tidak dianggap keluar).
//  - Status : keluar zona langsung dihitung sebagai akhir; status empty tidak dicek (sensor muatan baru terbaca
//             empty ±2 menit setelah unit keluar, dan pada beberapa unit tidak pernah terbaca).
//  - Batal  : bacaan empty ≥ 5 berturut-turut sebelum tiba di zona (kedipan sensor, bukan muatan sungguhan)
//             atau lebih dari 12 jam tanpa tiba. Antrian boleh berjam-jam (DT10250 antri ±5 jam).
import type { Point } from '@/types/telemetry.types'

export const HAUL_START_SEGMENT = 'Km 01+000 - 02+000'
export const HAUL_END_ZONES = ['Stockpile Emplacement', 'TLS2 Zona 4']
export const HAUL_WAIT_SEGMENT = 'Jembatan Timbang Km 01'

const ENTRY_CONFIRM = 3
const STOP_CONFIRM = 2
const EXIT_CONFIRM = 3
const EMPTY_ABORT = 5
const MAX_SECONDS = 12 * 3600

export type HaulPoint = Pick<Point, 'event_time' | 'segment_name' | 'vessel_status' | 'speed_kph'>

export interface HaulCycle {
  /** indeks pada daftar masukan: mulai antri (speed 0), masuk Km 01–02, dan titik pertama di luar zona */
  startIdx: number
  entryIdx: number
  endIdx: number
  start: string
  end: string
  /** durasi = mulai antri → keluar zona */
  seconds: number
  /** true bila data berakhir saat unit masih di dalam zona (belum keluar) */
  ongoing: boolean
}

const sec = (a: string, b: string) => (new Date(b).getTime() - new Date(a).getTime()) / 1000
const inEnd = (p: HaulPoint) => !!p.segment_name && HAUL_END_ZONES.includes(p.segment_name)
const inQueueArea = (p: HaulPoint) => p.segment_name === HAUL_START_SEGMENT || p.segment_name === HAUL_WAIT_SEGMENT

/** rows harus berurutan menurut waktu. */
export function findHaulCycles(rows: HaulPoint[]): HaulCycle[] {
  const out: HaulCycle[] = []
  let i = 0
  while (i < rows.length) {
    // 1) cari titik masuk: run loaded berturut-turut di Km 01+000 - 02+000
    let run = 0
    let entry = -1
    for (; i < rows.length; i++) {
      const p = rows[i]
      if (p.segment_name === HAUL_START_SEGMENT && p.vessel_status === 'loaded') {
        run++
        if (run === ENTRY_CONFIRM) {
          entry = i - ENTRY_CONFIRM + 1
          i++
          break
        }
      } else if (p.segment_name !== HAUL_WAIT_SEGMENT) {
        run = 0
      }
    }
    if (entry < 0) break

    // 2) ikuti: mulai antri → meninggalkan area antri → tiba di zona → keluar zona
    let stop = -1
    let zeroRun = 0
    let arrived = -1
    let lastIn = -1
    let outside = 0
    let emptyRun = 0
    let end = -1
    let aborted = false
    for (let j = entry; j < rows.length; i = ++j) {
      const p = rows[j]
      if (arrived < 0 && sec(rows[entry].event_time, p.event_time) > MAX_SECONDS) {
        aborted = true
        break
      }
      if (stop < 0) {
        if (inQueueArea(p) && p.speed_kph === 0) {
          zeroRun++
          if (zeroRun >= STOP_CONFIRM) stop = j - STOP_CONFIRM + 1
        } else {
          zeroRun = 0
        }
      }
      if (inEnd(p)) {
        if (arrived < 0) arrived = j
        lastIn = j
        outside = 0
        emptyRun = 0
      } else if (arrived >= 0) {
        outside++
        if (outside >= EXIT_CONFIRM) {
          end = lastIn + 1
          i = j + 1
          break
        }
      } else {
        emptyRun = p.vessel_status === 'empty' ? emptyRun + 1 : 0
        if (emptyRun >= EMPTY_ABORT) {
          aborted = true
          break
        }
      }
    }
    if (aborted) continue // lanjut mencari titik masuk berikutnya dari titik ini
    if (arrived < 0) break // data habis sebelum tiba di zona

    const ongoing = end < 0
    const endIdx = ongoing ? rows.length - 1 : end
    const startIdx = stop >= 0 ? stop : entry
    out.push({
      startIdx,
      entryIdx: entry,
      endIdx,
      start: rows[startIdx].event_time,
      end: rows[endIdx].event_time,
      seconds: sec(rows[startIdx].event_time, rows[endIdx].event_time),
      ongoing,
    })
  }
  return out
}

/** 2242 → "37 mnt 22 dtk"; ≥ 1 jam → "1 j 05 mnt". */
export function formatSeconds(total: number): string {
  const s = Math.round(total)
  if (s < 60) return `${s} dtk`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} mnt ${String(s % 60).padStart(2, '0')} dtk`
  return `${Math.floor(m / 60)} j ${String(m % 60).padStart(2, '0')} mnt`
}
