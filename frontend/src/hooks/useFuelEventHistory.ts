import { useQuery } from '@tanstack/react-query'

import { listFuelEventsByRange } from '@/services/api/fuel.api'
import type { FilterValues } from '@/types/filter.types'

import { useProjectKey } from './useProjectKey'

/** Kejadian BBM satu rentang tanggal (filter dari URL: from, to, shift, equipment). */
export function useFuelEventHistory(f: FilterValues) {
  const { pid, key } = useProjectKey('fuel-history', f)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!f.from && !!f.to,
    queryFn: () => listFuelEventsByRange({ from: f.from, to: f.to, shift: f.shift, equipmentId: f.equipment }),
  })
}
