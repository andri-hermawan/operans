// Tipe ekspor Excel (fungsinya ada di utils/excel.ts).

export type ExcelCell = string | number | boolean | Date | null | undefined

export interface ExcelColumn<T> {
  header: string
  /** Nilai sel dari satu baris data. */
  value: (row: T) => ExcelCell
  /** Format angka/tanggal Excel, mis. '#,##0.00' atau 'dd-mm-yyyy'. */
  numFmt?: string
  width?: number
}

export interface ExcelSheet<T> {
  name: string
  columns: ExcelColumn<T>[]
  rows: T[]
}
