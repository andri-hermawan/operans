// Nilai filter disimpan di URL (?date=...&shift=...) → bisa dibagikan/di-bookmark, tetap saat halaman dimuat ulang,
// dan halaman lain cukup membuat link dengan parameter yang sama (mis. dari alert ke riwayat posisi).
import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'

import type { FilterValues } from '@/types/filter.types'

export function useFilters(defaults: FilterValues = {}): [FilterValues, (patch: FilterValues) => void] {
  const [params, setParams] = useSearchParams()

  // defaults dibandingkan menurut isinya (objek baru tiap render tidak memicu hitung ulang)
  const defaultsKey = JSON.stringify(defaults)
  const values = useMemo(() => {
    const v: FilterValues = JSON.parse(defaultsKey)
    params.forEach((val, key) => {
      v[key] = val || undefined
    })
    return v
  }, [params, defaultsKey])

  const update = useCallback(
    (patch: FilterValues) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [k, v] of Object.entries(patch)) {
            if (v === undefined || v === '') next.delete(k)
            else next.set(k, v)
          }
          return next
        },
        { replace: true },
      )
    },
    [setParams],
  )

  return [values, update]
}
