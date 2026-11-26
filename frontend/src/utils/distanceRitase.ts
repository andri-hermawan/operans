// Ritase berdasarkan jarak (siklus penuh unit) dari titik telemetri per unit.
//
// Satu ritase = unit mulai bergerak di segmen A dengan status muatan S (empty atau loaded, tergantung kondisi awal),
// status berubah ke kebalikannya (mengambil muatan / dumping), lalu unit kembali ke segmen A dengan status S lagi.
// Contoh: empty → loaded → empty, atau (awal shift sudah membawa muatan) loaded → empty → loaded.
// Siklus yang belum kembali ke segmen A tidak dihitung.
//
//  - Start    : bacaan pertama unit bergerak (speed > 0) di segmen Km (nama berawalan "Km") saat status muatan sudah
//               terkonfirmasi. Bila unit awalnya di tempat non-Km (Stock ROM, CSA, Trase Workshop, dll.) atau statusnya masih
//               unknown, start ditunda sampai unit masuk segmen Km pertama dengan status terbaca; segmen itu menjadi segmen A.
//  - End      : bacaan pertama kembali di segmen A dengan status S, sesudah status kebalikannya terkonfirmasi.
//  - Status   : bacaan unknown diabaikan (status terakhir yang sudah terbaca tetap berlaku); status baru dianggap sungguhan
//               bila bertahan ≥ 2 menit (sensor muatan sering berkedip loaded/empty, bahkan saat unit melaju).
//  - Jarak    : jarak tempuh siklus (kecepatan × waktu) harus ≥ MIN_RITASE_KM. Siklus pendek (putar-putar di area stockpile /
//               emplacement, status muatan masih berkedip setelah dumping) bukan ritase hauling dan tidak dihitung.
import type { Point } from '@/types/telemetry.types'

/** status muatan baru dianggap sungguhan bila bertahan minimal selama ini (detik) */
const MIN_STATE_SECONDS = 120
/** jarak tempuh minimal satu ritase (km): ritase hauling asli ±50–60 km (data 26/11: ±60 km); putaran di area stockpile hanya 2–5 km */
export const MIN_RITASE_KM = 40
/** selisih waktu antar titik dibatasi agar jeda data tidak dihitung sebagai jarak */
const MAX_GAP_SECONDS = 30
/** segmen jalan hauling, mis. "Km 08+000 - 09+000" */
const KM_SEGMENT = /^Km \d/

export type RitasePoint = Pick<Point, 'event_time' | 'segment_name' | 'vessel_status' | 'speed_kph' | 'operator_name' | 'shift_code'>

export interface DistanceRitase {
  startIdx: number
  endIdx: number
  start: string
  end: string
  seconds: number
  /** jarak tempuh (km) */
  km: number
  /** segmen A: tempat ritase dimulai dan selesai */
  segment: string
  operator: string | null
  shiftCode: string | null
  /** status muatan saat ritase dimulai (dan saat selesai) */
  status: 'loaded' | 'empty'
}

const sec = (a: string, b: string) => (new Date(b).getTime() - new Date(a).getTime()) / 1000

/**
 * Status muatan terkonfirmasi di tiap titik (null sebelum ada status yang terkonfirmasi).
 * Sensor muatan berkedip loaded/empty selama detik sampai ±1 menit (saat memuat, melaju, atau melewati lokasi tertentu),
 * sedangkan status asli bertahan menit sampai puluhan menit. Karena itu bacaan dikelompokkan menjadi rentang status yang sama;
 * rentang yang lebih pendek dari MIN_STATE_SECONDS dianggap kedipan dan diserap ke status di sekitarnya.
 */
function confirmedStatus(rows: RitasePoint[]): (('loaded' | 'empty') | null)[] {
  type Run = { status: 'loaded' | 'empty'; first: number; t0: number; t1: number }
  const runs: Run[] = []
  rows.forEach((p, i) => {
    const v = p.vessel_status
    if (v !== 'loaded' && v !== 'empty') return
    const t = new Date(p.event_time).getTime()
    const last = runs[runs.length - 1]
    if (last && last.status === v) last.t1 = t
    else runs.push({ status: v, first: i, t0: t, t1: t })
  })
  // buang kedipan, lalu gabungkan rentang bersebelahan yang statusnya sama
  const kept: Run[] = []
  for (const r of runs) {
    if ((r.t1 - r.t0) / 1000 < MIN_STATE_SECONDS) continue
    const last = kept[kept.length - 1]
    if (last && last.status === r.status) last.t1 = r.t1
    else kept.push({ ...r })
  }
  const out: (('loaded' | 'empty') | null)[] = []
  let k = -1
  for (let i = 0; i < rows.length; i++) {
    while (k + 1 < kept.length && kept[k + 1].first <= i) k++
    out.push(k >= 0 ? kept[k].status : null)
  }
  return out
}

/** rows harus berurutan menurut waktu. */
export function findDistanceRitases(rows: RitasePoint[]): DistanceRitase[] {
  const conf = confirmedStatus(rows)
  const out: DistanceRitase[] = []
  let cursor = 0
  while (cursor < rows.length) {
    // 1) start: bacaan bergerak pertama di segmen yang jelas, saat status muatan sudah terkonfirmasi
    let s = -1
    for (let i = cursor; i < rows.length; i++) {
      const p = rows[i]
      if (conf[i] && (p.speed_kph ?? 0) > 0 && p.segment_name && KM_SEGMENT.test(p.segment_name)) {
        s = i
        break
      }
    }
    if (s < 0) break
    const anchor = rows[s].segment_name as string
    const status = conf[s] as 'loaded' | 'empty'
    const opposite = status === 'empty' ? 'loaded' : 'empty'

    // 2) status kebalikan terkonfirmasi, lalu 3) kembali ke segmen A dengan status awal
    let l = -1
    for (let i = s + 1; i < rows.length; i++) {
      if (conf[i] === opposite) {
        l = i
        break
      }
    }
    if (l < 0) break
    let e = -1
    for (let i = l + 1; i < rows.length; i++) {
      if (conf[i] === status && rows[i].segment_name === anchor) {
        e = i
        break
      }
    }
    if (e < 0) break

    let km = 0
    for (let i = s + 1; i <= e; i++) {
      const dt = Math.min(sec(rows[i - 1].event_time, rows[i].event_time), MAX_GAP_SECONDS)
      km += ((rows[i].speed_kph ?? 0) * dt) / 3600
    }
    cursor = e // titik kembali = awal ritase berikutnya (juga bila siklus ini terlalu pendek dan dibuang)
    if (km < MIN_RITASE_KM) continue

    let operator = rows[s].operator_name
    for (let i = s; !operator && i <= e; i++) operator = rows[i].operator_name
    out.push({
      startIdx: s,
      endIdx: e,
      start: rows[s].event_time,
      end: rows[e].event_time,
      seconds: sec(rows[s].event_time, rows[e].event_time),
      km,
      segment: anchor,
      operator: operator ?? null,
      shiftCode: rows[s].shift_code,
      status,
    })
  }
  return out
}
