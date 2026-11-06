import { useQuery } from '@tanstack/react-query'

import { getExecutiveEmplacement } from '@/services/api/report.api'

import { useProjectKey } from './useProjectKey'

/** Produksi & inventory emplacement per partner untuk `date` (harian, MTD, YTD), dari stok emplacement. */
export const useExecutiveEmplacement = (date?: string) => {
  const { pid, key } = useProjectKey('report', 'executive-emplacement', date)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!date,
    queryFn: () => getExecutiveEmplacement(pid!, date!),
  })
}
