// Ritase berdasarkan jarak untuk BANYAK unit pada satu tanggal operasional. Datanya sama dengan Position History
// (titik telemetry per unit); aturan di utils/distanceRitase.ts. Unit diproses beberapa sekaligus agar tidak membebani server.
import { useEffect, useMemo, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'

import { listPoints } from '@/services/api/telemetry.api'
import type { Equipment } from '@/types/equipment.types'
import { findDistanceRitases, type DistanceRitase } from '@/utils/distanceRitase'

import { useProjectKey } from './useProjectKey'

const CONCURRENCY = 4

export interface UnitRitase {
  equipment: Equipment
  ritases: DistanceRitase[]
  error?: string
  truncated?: boolean
}

export function useDistanceRitaseReport(date: string | undefined, shiftCode: string | undefined, units: Equipment[]) {
  const qc = useQueryClient()
  const { pid } = useProjectKey()
  const [state, setState] = useState<{ key: string; results: Map<string, UnitRitase> }>({ key: '', results: new Map() })
  const unitsKey = units.map((u) => u.id).join(',')
  const runKey = `${pid}|${date}|${shiftCode ?? ''}|${unitsKey}`

  useEffect(() => {
    if (!pid || !date || !units.length) return
    let cancelled = false
    const queue = [...units]
    const done = new Map<string, UnitRitase>()
    const push = () => !cancelled && setState({ key: runKey, results: new Map(done) })

    const work = async () => {
      for (let u = queue.shift(); u && !cancelled; u = queue.shift()) {
        try {
          const res = await qc.fetchQuery({
            queryKey: [pid, 'haul-queue-points', date, shiftCode, u.id], // cache sama dengan laporan antrian
            staleTime: 5 * 60_000,
            queryFn: () => listPoints({ date, shift: shiftCode, equipmentId: u.id }),
          })
          const rows = (res.data ?? []).filter((p) => p.latitude != null)
          done.set(u.id, { equipment: u, ritases: findDistanceRitases(rows), truncated: res.truncated })
        } catch (e) {
          done.set(u.id, { equipment: u, ritases: [], error: e instanceof Error ? e.message : 'gagal memuat' })
        }
        push()
      }
    }
    void Promise.all(Array.from({ length: Math.min(CONCURRENCY, units.length) }, work))
    return () => {
      cancelled = true
    }
    // units dibandingkan menurut isinya (unitsKey), bukan referensi array
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pid, date, shiftCode, unitsKey, qc, runKey])

  const results = state.key === runKey ? state.results : new Map<string, UnitRitase>()
  return useMemo(
    () => ({ results, done: results.size, total: units.length, loading: results.size < units.length }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [state, units.length, runKey],
  )
}
