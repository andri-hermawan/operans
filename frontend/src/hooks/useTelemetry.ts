import { listActivity, listPoints, listSegmentSpeeds } from '@/services/api/telemetry.api'
import type { Slot } from '@/types/telemetry.types'

import { useSlotQuery } from './useSlotQuery'

export const usePoints = (slot: Slot, opts: { stepSeconds?: number; enabled?: boolean } = {}) =>
  useSlotQuery('points', listPoints, slot, { step_seconds: opts.stepSeconds }, opts.enabled ?? true)
export const useSegmentSpeeds = (slot: Slot) => useSlotQuery('segment-speeds', listSegmentSpeeds, slot)
export const useActivity = (slot: Slot, enabled = true) => useSlotQuery('activity', listActivity, slot, {}, enabled)
