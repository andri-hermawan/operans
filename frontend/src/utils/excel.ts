// Ekspor Excel. Semua halaman memakai fungsi ini (bukan menulis ExcelJS sendiri).
// ExcelJS dimuat saat dibutuhkan saja agar halaman pertama tetap ringan.
import type { ExcelSheet } from '@/types/excel.types'

/** Buat workbook (bisa banyak sheet) lalu unduh sebagai fileName.xlsx. */
export async function downloadExcel(fileName: string, sheets: ExcelSheet<never>[] | ExcelSheet<unknown>[]): Promise<void> {
  const { Workbook } = await import('exceljs')
  const wb = new Workbook()
  for (const sheet of sheets as ExcelSheet<unknown>[]) {
    const ws = wb.addWorksheet(sheet.name.slice(0, 31))
    ws.columns = sheet.columns.map((c) => ({ header: c.header, width: c.width ?? Math.max(12, c.header.length + 4), style: c.numFmt ? { numFmt: c.numFmt } : {} }))
    for (const row of sheet.rows) ws.addRow(sheet.columns.map((c) => c.value(row) ?? null))
    const head = ws.getRow(1)
    head.font = { bold: true, color: { argb: 'FFFFFFFF' } }
    head.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: 'FF0B4F8C' } }
    ws.views = [{ state: 'frozen', ySplit: 1 }]
  }
  const buf = await wb.xlsx.writeBuffer()
  saveBlob(new Blob([buf], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), `${fileName}.xlsx`)
}

/** Simpan Blob sebagai file unduhan (juga untuk template dari server). */
export function saveBlob(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  a.click()
  URL.revokeObjectURL(url)
}
