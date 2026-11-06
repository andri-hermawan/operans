import { listFuelEvents } from '@/services/api/fuel.api'
import type { Slot } from '@/types/telemetry.types'

import { useSlotQuery } from './useSlotQuery'

export const useFuelEvents = (slot: Slot, enabled = true) => useSlotQuery('fuel-events', listFuelEvents, slot, {}, enabled)
