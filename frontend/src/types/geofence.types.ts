// Kejadian masuk/keluar area (geofence) dari layanan IoT.

export interface GeofenceEvent {
  id: number
  equipment_id: string
  event_type: 'enter' | 'exit'
  event_time: string
  shift_code: string | null
  segment_name: string | null
  location_category: string | null
  latitude: number | null
  longitude: number | null
  speed_kph: number | null
  vessel_status: string | null
}
