import { useQuery } from '@tanstack/react-query'

import { searchDocuments } from '@/services/api/aiDocument.api'
import type { DocumentFilters } from '@/types/aiDocument.types'

import { useProjectKey } from './useProjectKey'

export const MIN_SEARCH_CHARS = 2

/** Pencarian kata kunci di dalam isi dokumen; berjalan bila kata kunci cukup panjang. */
export function useDocumentSearch(query: string, filters: DocumentFilters) {
  const q = query.trim()
  const { pid, key } = useProjectKey('ai-search', q, filters)
  return useQuery({
    queryKey: key,
    enabled: !!pid && q.length >= MIN_SEARCH_CHARS,
    queryFn: ({ signal }) => searchDocuments(q, filters, signal),
  })
}
