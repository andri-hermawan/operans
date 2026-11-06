// Kejadian isi/turun BBM dari layanan IoT.

export interface FuelEvent {
  id: number
  equipment_id: string
  event_type: 'increase' | 'decrease'
  event_time: string
  fuel_delta_liter: number
  fuel_volume_liter: number | null
  description: string | null
  alert_id: number | null
}

/** Baris laporan Fuel History: kejadian BBM + kode unit. */
export type FuelEventRow = FuelEvent & { code: string }
