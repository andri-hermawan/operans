// Jenis alert (Alert Category) dan aturan ambang per project/zona (core-api).

export interface AlertType {
  code: string
  legacy_code: string
  name: string
  description: string
  uses_duration: boolean
  uses_speed_limit: boolean
  uses_fuel_drop: boolean
  default_duration_minutes: number | null
  default_speed_limit_kph: number | null
  default_fuel_drop_liters: number | null
}

export interface AlertRule {
  id: string
  alert_type: string
  alert_type_name: string
  zone_name: string | null
  duration_minutes: number | null
  speed_limit_kph: number | null
  fuel_drop_liters: number | null
  description: string | null
  is_active: boolean
  recipients: { user_id: string; name: string; phone_number: string | null }[]
  updated_at: string
}
