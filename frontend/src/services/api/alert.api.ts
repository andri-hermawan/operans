// Alert abnormal (layanan IoT).
import type { AlertRow } from '@/types/alert.types'
import type { Slot, SlotExtra } from '@/types/telemetry.types'

import { listBySlot } from './slot.api'

export const listAlerts = (slot: Slot, extra?: SlotExtra) => listBySlot<AlertRow>('/api/v1/alerts', slot, extra)
