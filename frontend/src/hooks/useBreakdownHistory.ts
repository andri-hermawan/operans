import { useQuery } from '@tanstack/react-query'

import { breakdownApi } from '@/services/api/daily.api'
import type { FilterValues } from '@/types/filter.types'

import { useCodeOf } from './useCodeOf'
import { useProjectKey } from './useProjectKey'
import { useShifts } from './useShifts'

/** Catatan breakdown satu rentang tanggal (filter dari URL: from, to, shift, equipment) — sama dengan Fuel History. */
export function useBreakdownHistory(f: FilterValues) {
  const { pid, key } = useProjectKey('breakdown-history', f)
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  const shiftId = f.shift ? shifts.find((s) => s.shift_code === f.shift)?.id : undefined
  const equipmentCode = f.equipment ? codeOf(f.equipment) : undefined
  return useQuery({
    queryKey: [...key, shiftId, equipmentCode],
    enabled: !!pid && !!f.from && !!f.to && (!f.shift || !!shiftId),
    queryFn: () => breakdownApi.range({ projectId: pid!, from: f.from, to: f.to, shiftId, equipmentCode }),
  })
}
