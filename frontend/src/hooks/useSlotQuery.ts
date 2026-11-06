import { useQuery } from '@tanstack/react-query'

import type { TruncatedList } from '@/types/api.types'
import type { Slot, SlotExtra } from '@/types/telemetry.types'

import { useProjectKey } from './useProjectKey'

/** Dasar semua hook data IoT yang disaring tanggal operasional + shift + unit. Menunggu tanggal terisi. */
export const useSlotQuery = <T>(
  name: string,
  fetcher: (slot: Slot, extra: SlotExtra) => Promise<TruncatedList<T>>,
  slot: Slot,
  extra: SlotExtra = {},
  enabled = true,
) => {
  const { pid, key } = useProjectKey(name, slot, extra)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!slot.date && enabled,
    queryFn: () => fetcher(slot, extra),
  })
}
