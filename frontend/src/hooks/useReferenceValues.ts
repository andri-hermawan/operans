import { useQuery } from '@tanstack/react-query'

import { listReferenceValues } from '@/services/api/referenceValue.api'

export function useReferenceValues(group?: string) {
  return useQuery({
    queryKey: ['reference-values', group],
    enabled: !!group,
    staleTime: 30 * 60_000,
    queryFn: () => listReferenceValues(group!),
  })
}
