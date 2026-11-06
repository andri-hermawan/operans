// Daftar data IoT yang disaring tanggal operasional + shift + unit ("slot"). Dipakai telemetry, alert, geofence, fuel.
import { iot } from '@/services/http/client'
import type { TruncatedList } from '@/types/api.types'
import type { Slot, SlotExtra } from '@/types/telemetry.types'

export const listBySlot = <T>(path: string, slot: Slot, extra: SlotExtra = {}) =>
  iot.get<TruncatedList<T>>(path, { params: { date: slot.date, shift: slot.shift, equipment_id: slot.equipmentId, ...extra } })
