import { useQuery } from '@tanstack/react-query'

import { getZones } from '@/services/api/zone.api'

import { useProjectKey } from './useProjectKey'

/** Peta area project yang berlaku (GeoJSON dari core-api). */
export function useZones() {
  const { pid, key } = useProjectKey('zones')
  return useQuery({
    queryKey: key,
    enabled: !!pid,
    staleTime: 30 * 60_000,
    queryFn: () => getZones(pid!),
  })
}
