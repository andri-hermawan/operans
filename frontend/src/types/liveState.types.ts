// Kondisi terakhir unit (peta live) dari layanan IoT + status sambungan WebSocket.
import type { AlertRow } from './alert.types'
import type { GeofenceEvent } from './geofence.types'

export interface LiveState {
  equipment_id: string
  device_id: string
  event_time: string
  received_at: string
  operational_date: string | null
  shift_code: string | null
  latitude: number | null
  longitude: number | null
  speed_kph: number | null
  heading_deg: number | null
  is_engine_on: boolean | null
  equipment_status: string | null // running | idle | stop | offline
  vessel_status: string | null // empty | loaded | unknown
  gsm_signal_level: number | null
  mileage_m: number | null
  is_inside_geofence: boolean | null
  location_category: string | null
  segment_name: string | null
  fuel_volume_liter: number | null
  fuel_percent: number | null
  operator_name: string | null
  is_breakdown: boolean | null
  breakdown_description: string | null
  alert_count: number
}

export type LiveStatus = 'connecting' | 'live' | 'offline'

/** Penerima kejadian baru dari WebSocket (opsional per halaman). */
export interface LiveHandlers {
  onAlert?: (a: AlertRow) => void
  onGeofence?: (g: GeofenceEvent) => void
}
