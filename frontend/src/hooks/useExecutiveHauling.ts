import { useQuery } from '@tanstack/react-query'

import { getExecutiveHauling } from '@/services/api/report.api'

import { useProjectKey } from './useProjectKey'

/** Produksi hauling per customer untuk `date` (harian, MTD, YTD), dari tiket timbangan project aktif. */
export const useExecutiveHauling = (date?: string) => {
  const { pid, key } = useProjectKey('report', 'executive-hauling', date)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!date,
    queryFn: () => getExecutiveHauling(pid!, date!),
  })
}
