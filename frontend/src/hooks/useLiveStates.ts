import { useQuery } from '@tanstack/react-query'

import { listLiveStates } from '@/services/api/liveState.api'

import { useProjectKey } from './useProjectKey'

const LIVE_KEY = 'live-states'
export const liveKey = (pid?: string) => [pid, LIVE_KEY] as const

/** Kondisi terakhir semua unit project (diperbarui WebSocket lewat useLiveSocket). */
export function useLiveStates() {
  const { pid, key } = useProjectKey(LIVE_KEY)
  return useQuery({
    queryKey: key,
    enabled: !!pid,
    staleTime: Infinity, // perubahan datang dari WebSocket
    queryFn: listLiveStates,
  })
}
