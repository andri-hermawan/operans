import { useQuery } from '@tanstack/react-query'

import { listGeoImports } from '@/services/api/geoImport.api'

import { useProjectKey } from './useProjectKey'

/** Riwayat unggahan peta area project aktif. */
export function useGeoImports() {
  const { pid, key } = useProjectKey('geo-imports')
  return useQuery({ queryKey: key, enabled: !!pid, queryFn: () => listGeoImports(pid!) })
}
