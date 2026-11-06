import { useQuery } from '@tanstack/react-query'

import { breakdownApi } from '@/services/api/daily.api'

import { useProjectKey } from './useProjectKey'

/** Catatan breakdown satu tanggal operasional (+ satu shift bila shiftId diisi). */
export function useEquipmentBreakdowns(date?: string, shiftId?: string) {
  const { pid, key } = useProjectKey('breakdowns', date, shiftId)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!date,
    queryFn: () => breakdownApi.list(pid!, date, shiftId),
  })
}
