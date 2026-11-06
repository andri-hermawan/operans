// Isi laporan Executive Summary Hauling (Operation Performance): produksi per customer — Daily / MTD / YTD.
// ACTUAL dari tiket timbangan (kg → ton). PLAN belum ada sumbernya → sementara 0 (isi lewat prop `plan` bila sudah ada).
import { Alert } from 'antd'

import type { ExecutiveHaulingRow, Cell, Line } from '@/types/report.types'

import { PERIODS, sumCells } from '@/utils/performance'
import PerformanceTable from './PerformanceTable'

interface Props {
  rows: ExecutiveHaulingRow[]
  /** Rencana (ton) per customer: { SWE: { daily, mtd, ytd } }. Belum tersedia → dianggap 0. */
  plan?: Record<string, { daily: number; mtd: number; ytd: number }>
}

/** Urutan baris seperti laporan Excel; customer lain di tiket ditambahkan di bawahnya agar total tetap benar. */
const CUSTOMERS = ['TBBE', 'SWE', 'SBL', 'DBU', 'UN', 'CPM', 'MME', 'TPB']

export default function ExecutiveHaulingReport({ rows, plan = {} }: Props) {
  const byCustomer = new Map(rows.map((r) => [r.customer, r]))
  const names = [...CUSTOMERS, ...rows.map((r) => r.customer).filter((c) => !CUSTOMERS.includes(c))]

  const lines: Line[] = names.map((n) => {
    const r = byCustomer.get(n)
    const p = plan[n]
    return {
      label: `Coal Hauling ${n}`,
      cells: {
        daily: { plan: p?.daily ?? 0, actual: (r?.daily_kg ?? 0) / 1000 },
        mtd: { plan: p?.mtd ?? 0, actual: (r?.mtd_kg ?? 0) / 1000 },
        ytd: { plan: p?.ytd ?? 0, actual: (r?.ytd_kg ?? 0) / 1000 },
      },
    }
  })
  const total: Line = {
    label: 'Total All Project',
    bold: true,
    cells: Object.fromEntries(PERIODS.map(({ key }) => [key, sumCells(lines, key)])) as Record<string, Cell>,
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Alert type="info" showIcon title="PLAN belum diisi (sementara 0) — ACTUAL dari tiket timbangan (customer), dalam ton." />
      <PerformanceTable sections={[{ title: 'Production', lines: [total, ...lines] }]} />
    </div>
  )
}
