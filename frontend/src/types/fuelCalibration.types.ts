// Kalibrasi sensor fuel per unit (core-api).

export interface FuelCalibration {
  equipment_id: string
  equipment_code: string
  calibration_type: 'lls' | 'percentage'
  tank_capacity_liters: number
  max_sensor_value: number
  is_linear: boolean
  point_count: number
  updated_at: string
}
