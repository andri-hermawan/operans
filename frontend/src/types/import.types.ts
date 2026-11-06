// Impor Excel ke core-api (ADR 0007) dan hasil pratinjaunya.

/** Jenis impor core-api (ADR 0007). */
export type ImportType =
  | 'operator-assignments'
  | 'equipment-activations'
  | 'equipment-breakdowns'
  | 'weighbridge-tickets'
  | 'stock-emplacements'
  | 'yearly-plans'
  | 'monthly-plans'
  | 'time-sheets'
  | 'ritases'
  | 'work-hour-notifications'
  | 'rain-slippery-logs'

export interface RowIssue {
  row?: number
  column?: string
  message: string
}
export interface ImportResult {
  total_rows: number
  valid_rows: number
  invalid_rows: number
  duplicate_rows: number
  to_create: number
  to_update: number
  saved: boolean
  errors: RowIssue[]
  warnings: RowIssue[]
}
