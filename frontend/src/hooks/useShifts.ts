import { useQuery } from '@tanstack/react-query'

import { listShifts } from '@/services/api/shift.api'

import { useProjectKey } from './useProjectKey'

/** Shift project aktif, urut sequence. */
export function useShifts() {
  const { pid, key } = useProjectKey('shifts')
  return useQuery({
    queryKey: key,
    enabled: !!pid,
    staleTime: 10 * 60_000,
    queryFn: async () => (await listShifts(pid!)).sort((a, b) => a.sequence - b.sequence),
  })
}
