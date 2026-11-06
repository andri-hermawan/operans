// Alat GPS (iot.devices, layanan IoT).

export interface Device {
  id: string
  imei: string
  name: string | null
  equipment_id: string | null
  equipment_code: string | null
  model: string | null
  fuel_source: string | null
  sim_number: string | null
  provider_name: string | null
  has_api_key: boolean
  is_active: boolean
  last_event_at: string | null
}
