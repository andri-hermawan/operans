// Input harian core-api (Record / Upload, ADR 0007): operator, aktivasi unit, breakdown, tiket timbangan.

export interface OperatorAssignment {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  equipment_code: string
  operator_name: string
  description: string | null
}

export interface EquipmentActivation {
  id: number
  equipment_id: string
  equipment_code: string
  activation_date: string
  status: 'active' | 'inactive'
  description: string | null
}

export interface EquipmentBreakdown {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  equipment_code: string
  status: string
  category_code: string
  category_label: string
  started_at: string | null
  ended_at: string | null
  duration_hours: number | null
  repair_status_code: string
  repair_status_label: string
  description: string | null
  location: string | null
}

export interface WeighbridgeTicket {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  ticket_no: string
  equipment_code: string
  product: string | null
  gross_kg: number
  tare_kg: number
  net_kg: number
  recipient: string | null
  customer: string | null
  transporter: string | null
  gross_at: string | null
  tare_at: string | null
  gross_operator: string | null
  tare_operator: string | null
  description: string | null
  location: string | null
}

/** Stok cargo harian per project dan partner (ton), satu baris per partner per tanggal. Kolom total disimpan apa adanya dari input. */
export interface StockEmplacement {
  id: number
  project_id: string
  partner: string
  operational_date: string
  stock_cargo_initial: number | null
  stock_cargo_scrapper: number | null
  stock_cargo_crusher: number | null
  info: string | null
  shift_1: number | null
  shift_2: number | null
  hauling_cargo: number | null
  total_cargo_initial: number | null
  tls_out_trainset_1: number | null
  tls_out_trainset_2: number | null
  tls_out_trainset_3: number | null
  tls_out_trainset_4: number | null
  tls_out_trainset_5: number | null
  total_tls_out: number | null
  remaining_cargo: number | null
}

/** Time Sheet: HM (hour meter) awal/akhir unit per tanggal + shift. HM total disimpan apa adanya dari input. */
export interface TimeSheet {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  equipment_code: string
  operator_nik: string | null
  operator_name: string | null
  hm_start: number | null
  hm_end: number | null
  hm_total: number | null
  remark: string | null
}

/** Ritase: hasil angkut per jam kerja (loader + hauler, lokasi, material, produksi). */
export interface Ritase {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  time_slot: string
  loader_code: string
  loader_model: string | null
  loader_operator_nik: string | null
  loader_operator_name: string | null
  loader_partner: string | null
  hauler_code: string
  hauler_model: string | null
  hauler_operator_nik: string | null
  hauler_operator_name: string | null
  hauler_partner: string | null
  block: string | null
  location: string | null
  dumping: string | null
  material: string | null
  sub_material: string | null
  seam: string | null
  phase: string | null
  distance: number | null
  ritase: number | null
  production: number | null
  production_adjusted: number | null
  distance_volume: number | null
  productivity_factor: number | null
}

/** WH & Notification: kejadian jam kerja unit (standby, dll). */
export interface WorkHourNotification {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  equipment_code: string
  category: string
  sub_category: string | null
  started_at: string
  ended_at: string | null
  total_hours: number | null
  dedicated: string | null
  remark: string | null
}

/** Rain & Slippery: hujan, tunggu licin, dan licin per lokasi + shift. */
export interface RainSlipperyLog {
  id: number
  operational_date: string
  shift_code: string
  shift_name: string
  location: string | null
  rain_start_at: string | null
  rain_stop_at: string | null
  rain_hours: number | null
  wait_slippery_start_at: string | null
  wait_slippery_stop_at: string | null
  wait_slippery_hours: number | null
  slippery_start_at: string | null
  slippery_stop_at: string | null
  slippery_hours: number | null
  rainfall_mm: number | null
}
