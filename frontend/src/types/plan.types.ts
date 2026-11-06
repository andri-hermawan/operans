// Rencana produksi (core-api, ADR 0014): Plan Yearly (per bulan) dan Plan Monthly (per tanggal).

export type MonthKey = 'jan' | 'feb' | 'mar' | 'apr' | 'may' | 'jun' | 'jul' | 'aug' | 'sep' | 'oct' | 'nov' | 'dec'

/** day_01 … day_31 (lama: T01 … T31). */
export type DayKey = `day_${string}`

/** Plan Yearly: satu baris per mitra, tahun, lokasi, deskripsi, dan nama. Total disimpan apa adanya (kosong → jumlah bulan). */
export type YearlyPlan = {
  id: number
  project_id: string
  partner: string
  plan_year: number
  location: string
  description: string
  plan_name: string
  total: number
  updated_at: string
} & Record<MonthKey, number>

/** Plan Monthly: satu baris per mitra, tahun, bulan, deskripsi, dan nama. Total disimpan apa adanya (kosong → jumlah tanggal). */
export type MonthlyPlan = {
  id: number
  project_id: string
  partner: string
  plan_year: number
  plan_month: number
  description: string
  plan_name: string
  uom: string
  total: number
  updated_at: string
} & Record<DayKey, number>
