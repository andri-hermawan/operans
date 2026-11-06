// Isi laporan Executive Summary Emplacement (Operation Performance), dari stok emplacement per partner:
//  · Emplacement Production = Total TLS Out (Daily / MTD / YTD)
//  · Emplacement Inventory  = Sisa Cargo pada tanggal laporan (hanya Daily)
// PLAN belum ada sumbernya → sementara 0.
import { Alert } from 'antd'

import type { ExecutiveEmplacementRow, Cell, Line, Period } from '@/types/report.types'

import { sumCells } from '@/utils/performance'
import PerformanceTable from './PerformanceTable'

interface Props {
  rows: ExecutiveEmplacementRow[]
}

/** Urutan baris seperti laporan Excel; partner lain di data stok ditambahkan di bawahnya agar total tetap benar. */
const PARTNERS = ['TBBE', 'SWE', 'SWE C', 'SWE S', 'SBL', 'DBU', 'UN', 'CPM', 'MME', 'TPB']

export default function ExecutiveEmplacementReport({ rows }: Props) {
  const byPartner = new Map(rows.map((r) => [r.partner, r]))
  const names = [...PARTNERS, ...rows.map((r) => r.partner).filter((p) => !PARTNERS.includes(p))]
  const cell = (actual = 0): Cell => ({ plan: 0, actual })

  const production: Line[] = names.map((n) => {
    const r = byPartner.get(n)
    return { label: `Coal ${n}`, cells: { daily: cell(r?.daily_production), mtd: cell(r?.mtd_production), ytd: cell(r?.ytd_production) } }
  })
  const inventory: Line[] = names.map((n) => ({ label: `Coal ${n}`, cells: { daily: cell(byPartner.get(n)?.inventory) } }))

  const totalOf = (label: string, lines: Line[], keys: readonly Period[]): Line => ({
    label,
    bold: true,
    cells: Object.fromEntries(keys.map((k) => [k, sumCells(lines, k)])) as Record<string, Cell>,
  })

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Alert type="info" showIcon title="PLAN belum diisi (sementara 0) — ACTUAL dari stok emplacement: Production = Total TLS Out, Inventory = Sisa Cargo (ton)." />
      <PerformanceTable
        sections={[
          { title: 'Emplacement Production', lines: [totalOf('Coal Train Load Set', production, ['daily', 'mtd', 'ytd']), ...production] },
          { title: 'Emplacement Inventory', lines: [totalOf('All Inventory', inventory, ['daily']), ...inventory] },
        ]}
      />
    </div>
  )
}
