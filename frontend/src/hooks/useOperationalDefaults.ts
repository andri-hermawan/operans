import { useEffect, useMemo, useState } from 'react'

import { operationalSlot } from '@/utils/time'

import { useShifts } from './useShifts'

/** Seberapa sering jam diperiksa untuk mengikuti pergantian shift/tanggal (milidetik). */
const CLOCK_MS = 30_000

/**
 * Tanggal operasional & shift yang sedang berjalan (dari shift project).
 * - `follow = false` (bawaan): dihitung sekali sebagai nilai awal filter, sehingga halaman yang sedang dikerjakan tidak
 *   berubah sendiri di tengah jalan.
 * - `follow = true`: ikut berganti otomatis saat shift/tanggal berganti (diperiksa tiap 30 detik). Dipakai halaman
 *   pemantauan live 24 jam yang kolom tanggal & shift-nya dikunci (Live Tracking, Geofencing, lonceng alert).
 */
export function useOperationalDefaults(follow = false): { date: string; shift?: string; ready: boolean } {
  const { data: shifts, isSuccess } = useShifts()
  const [clock, setClock] = useState(() => Date.now())
  useEffect(() => {
    if (!follow) return
    const t = setInterval(() => setClock(Date.now()), CLOCK_MS)
    return () => clearInterval(t)
  }, [follow])

  const slot = useMemo(() => operationalSlot(shifts ?? [], new Date(clock)), [shifts, clock])
  const date = slot.date
  const shift = slot.shift?.shift_code
  // objek yang sama selama tanggal & shift tidak berubah (pemakai tidak ikut dihitung ulang tiap 30 detik)
  return useMemo(() => ({ date, shift, ready: isSuccess }), [date, shift, isSuccess])
}
