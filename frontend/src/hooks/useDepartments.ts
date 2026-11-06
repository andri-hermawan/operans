import { useQuery } from '@tanstack/react-query'

import { listDepartments } from '@/services/api/aiDocument.api'

import { useProjectKey } from './useProjectKey'

/** Department (pengelompokan dokumen SOP) project aktif. */
export function useDepartments() {
  const { pid, key } = useProjectKey('ai-departments')
  return useQuery({ queryKey: key, enabled: !!pid, queryFn: listDepartments })
}
