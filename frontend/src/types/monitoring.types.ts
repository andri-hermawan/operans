// Data tampilan halaman pemantauan (popup titik/unit).
import type { ReactNode } from 'react'

export interface InfoData {
  time?: string | null
  speed?: number | null
  fuelLiter?: number | null
  fuelPercent?: number | null
  segment?: string | null
  status?: string | null
  vessel?: string | null
  engine?: boolean | null
  operator?: string | null
  lat?: number | null
  lon?: number | null
  extra?: { label: string; value: ReactNode }[]
}

/** Mode halaman riwayat: posisi atau BBM. */
export type HistoryMode = 'position' | 'fuel'
