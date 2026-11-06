import { useQuery } from '@tanstack/react-query'

import { listEquipment } from '@/services/api/equipment.api'

import { useProjectKey } from './useProjectKey'

/** Unit di project aktif. */
export function useEquipmentList(includeInactive = false) {
  const { pid, key } = useProjectKey('equipment', includeInactive)
  return useQuery({
    queryKey: key,
    enabled: !!pid,
    staleTime: 5 * 60_000,
    queryFn: () => listEquipment(pid!, includeInactive),
  })
}
