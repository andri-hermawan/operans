// Shift kerja project dan slot tanggal operasional (aturan: ADR 0004; fungsinya ada di utils/time.ts).

export interface Shift {
  id: string
  project_id: string
  shift_code: string // SHIFT_1
  shift_name: string // Shift 1
  start_time: string // HH:MM (jam lokal project)
  end_time: string
  sequence: number
}

export interface OperationalSlot {
  date: string // YYYY-MM-DD (tanggal dimulainya shift)
  shift?: Shift
}

/** Baris pengaturan shift (System Data, Shift), termasuk yang nonaktif. */
export interface ShiftRow {
  id: string
  shift_code: string
  shift_name: string
  start_time: string
  end_time: string
  sequence: number
  is_active: boolean
  updated_at: string
}
