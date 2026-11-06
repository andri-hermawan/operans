// Telemetri & ringkasan aktivitas unit (layanan IoT).
import type { Activity, HeatCell, Point, SegmentSpeed, Slot, SlotExtra } from '@/types/telemetry.types'

import { listBySlot } from './slot.api'

export const listPoints = (slot: Slot, extra?: SlotExtra) => listBySlot<Point>('/api/v1/telemetry/points', slot, extra)
export const listSegmentSpeeds = (slot: Slot, extra?: SlotExtra) => listBySlot<SegmentSpeed>('/api/v1/telemetry/segment-speeds', slot, extra)
export const listActivity = (slot: Slot, extra?: SlotExtra) => listBySlot<Activity>('/api/v1/activity-summary', slot, extra)

/** Kotak heatmap (sisi `cellM` meter) untuk bagian peta `bbox` — satu baris per kotak, bukan per titik. */
export const listHeatGrid = (slot: Slot, extra?: SlotExtra) => listBySlot<HeatCell>('/api/v1/telemetry/heat-grid', slot, extra)
