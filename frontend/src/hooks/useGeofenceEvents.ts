import { listGeofenceEvents } from '@/services/api/geofence.api'
import type { Slot } from '@/types/telemetry.types'

import { useSlotQuery } from './useSlotQuery'

export const useGeofenceEvents = (slot: Slot, segment?: string) =>
  useSlotQuery('geofence-events', listGeofenceEvents, slot, { segment }, !!segment) // tanpa area dipilih tidak mengambil data
