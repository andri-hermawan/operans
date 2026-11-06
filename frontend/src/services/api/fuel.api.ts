// Kejadian isi/turun BBM (layanan IoT).
import { iot } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { FuelEvent } from '@/types/fuel.types'
import type { Slot, SlotExtra } from '@/types/telemetry.types'

import { listBySlot } from './slot.api'

export const listFuelEvents = (slot: Slot, extra?: SlotExtra) => listBySlot<FuelEvent>('/api/v1/fuel-events', slot, extra)

/** Kejadian BBM untuk rentang tanggal operasional (laporan Fuel History). */
export const listFuelEventsByRange = async (p: { from?: string; to?: string; shift?: string; equipmentId?: string }) =>
  (await iot.get<ListResponse<FuelEvent>>('/api/v1/fuel-events', { params: { date_from: p.from, date_to: p.to, shift: p.shift, equipment_id: p.equipmentId } })).data
