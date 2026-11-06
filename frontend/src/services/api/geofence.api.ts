// Kejadian masuk/keluar area (layanan IoT).
import type { GeofenceEvent } from '@/types/geofence.types'
import type { Slot, SlotExtra } from '@/types/telemetry.types'

import { listBySlot } from './slot.api'

export const listGeofenceEvents = (slot: Slot, extra?: SlotExtra) => listBySlot<GeofenceEvent>('/api/v1/geofence-events', slot, extra)
