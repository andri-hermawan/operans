// Titik telemetri, ringkasan aktivitas, kecepatan per segmen, dan filter slot (tanggal + shift + unit).

export interface Point {
  id: number
  equipment_id: string
  event_time: string
  latitude: number | null
  longitude: number | null
  speed_kph: number | null
  heading_deg: number | null
  is_engine_on: boolean | null
  equipment_status: string | null
  vessel_status: string | null
  fuel_volume_liter: number | null
  fuel_percent: number | null
  fuel_delta_liter: number | null
  fuel_temperature_c: number | null
  segment_name: string | null
  location_category: string | null
  is_inside_geofence: boolean | null
  operator_name: string | null
  is_breakdown: boolean | null
  shift_code: string | null
  operational_date: string | null
}

export interface Activity {
  equipment_id: string
  running_hours: number
  running_empty_hours: number
  running_loaded_hours: number
  idle_hours: number
  idle_empty_hours: number
  idle_loaded_hours: number
  avg_speed: number
  avg_speed_empty: number
  avg_speed_loaded: number
  max_speed: number
  max_speed_empty: number
  max_speed_loaded: number
  mileage_km: number
  fuel_start_liter: number | null
  fuel_remaining_liter: number | null
  fuel_increase_liter: number
  fuel_decrease_liter: number
  fuel_burn_km_per_liter: number
}

export interface SegmentSpeed {
  segment_name: string
  avg_speed_empty: number | null
  avg_speed_loaded: number | null
  avg_speed: number | null
  points: number
}

/** Filter standar: tanggal operasional + shift + unit. */
export interface Slot {
  date?: string
  shift?: string
  equipmentId?: string
}

/** Parameter tambahan untuk daftar IoT per slot. */
export type SlotExtra = Record<string, string | number | undefined>

/** Baris Asset History: titik telemetri + kode unit. */
export type PointRow = Point & { code: string }

/** Satu kotak Speed Heatmap: ringkasan semua titik telemetry yang jatuh di kotak itu (dihitung di server). */
export interface HeatCell {
  latitude: number // titik tengah kotak
  longitude: number
  points: number // jumlah titik telemetry di kotak
  units: number // jumlah unit berbeda yang melintas
  avg_speed: number
  max_speed: number
  avg_speed_empty: number | null
  avg_speed_loaded: number | null
}

/** Bagian peta yang sedang terlihat: [minLon, minLat, maxLon, maxLat] + zoom. */
export interface MapView {
  bbox: [number, number, number, number]
  zoom: number
}
