import { listAlerts } from '@/services/api/alert.api'
import type { Slot } from '@/types/telemetry.types'

import { useSlotQuery } from './useSlotQuery'

/** Alert satu slot, atau rentang tanggal bila `dateTo` diisi. */
export const useAlerts = (slot: Slot & { type?: string; dateTo?: string }) =>
  useSlotQuery(
    'alerts',
    listAlerts,
    { date: slot.date, shift: slot.shift, equipmentId: slot.equipmentId },
    slot.dateTo ? { date: undefined, date_from: slot.date, date_to: slot.dateTo, type: slot.type } : { type: slot.type },
  )
