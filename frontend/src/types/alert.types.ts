// Alert dari layanan IoT.

export interface AlertRow {
  id: number
  equipment_id: string
  alert_type: string
  started_at: string
  ended_at: string | null
  operational_date: string | null
  shift_code: string | null
  threshold_value: number | null
  latitude: number | null
  longitude: number | null
  speed_kph: number | null
  is_engine_on: boolean | null
  vessel_status: string | null
  fuel_volume_liter: number | null
  fuel_percent: number | null
  fuel_delta_liter: number | null
  fuel_temperature_c: number | null
  segment_name: string | null
  location_category: string | null
  operator_name: string | null
}

/** Baris daftar alert: alert + kode unit, durasi, dan nama shift. */
export type AlertListRow = AlertRow & { code: string; duration: number | null; shift_name?: string }
