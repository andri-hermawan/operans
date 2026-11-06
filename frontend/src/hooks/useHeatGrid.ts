import { keepPreviousData, useQuery } from '@tanstack/react-query'

import { listHeatGrid } from '@/services/api/telemetry.api'
import type { MapView, Slot } from '@/types/telemetry.types'

import { useProjectKey } from './useProjectKey'

/** Sisi kotak (meter) menurut zoom peta: makin dekat makin halus. */
export const cellMetersForZoom = (zoom: number) => (zoom >= 19 ? 5 : zoom >= 18 ? 8 : zoom >= 17 ? 15 : zoom >= 16 ? 25 : zoom >= 15 ? 50 : zoom >= 14 ? 100 : zoom >= 13 ? 200 : 400)

const MAX_CELLS = 8000

/** Kotak Speed Heatmap untuk bagian peta yang terlihat. Data lama tetap tampil selagi data baru dimuat (tidak berkedip). */
export function useHeatGrid(slot: Slot, view?: MapView) {
  const cellM = view ? cellMetersForZoom(view.zoom) : undefined
  const bbox = view?.bbox.join(',')
  const { pid, key } = useProjectKey('heat-grid', slot, cellM, bbox)
  return {
    cellM,
    ...useQuery({
      queryKey: key,
      enabled: !!pid && !!slot.date && !!view,
      placeholderData: keepPreviousData,
      queryFn: () => listHeatGrid(slot, { cell_m: cellM, bbox, limit: MAX_CELLS }),
    }),
  }
}
