// Isi laporan Ritase Hauling per Customer: ritase (jumlah tiket timbangan) tiap unit, dipisah per customer, untuk satu tanggal operasional
// (semua shift atau satu shift). Daftar unit = seluruh unit Equipment Activation tanggal itu (tiket di-join by kode unit). Unit dibagi dua blok kiri-kanan (selalu berdampingan, jumlahnya dibagi dua) seperti laporan Excel; unit mitra (DT80) tidak ikut; di bawahnya total ritase & jumlah unit yang jalan.
import { Alert } from 'antd'

import type { RitaseCustomerCell } from '@/types/report.types'
import { brand } from '@/utils/colors'
import { formatNumber } from '@/utils/format'

interface Props {
  rows: RitaseCustomerCell[]
  /** Nama kolom total, mis. "Shift 1" atau "Semua Shift". */
  totalLabel: string
}

/** Urutan customer seperti laporan Excel; customer lain di tiket ditambahkan di belakangnya (urut abjad). */
const CUSTOMERS = ['SWE', 'SBL', 'DBU', 'UN', 'CPM', 'TPB']

const border = `1px solid ${brand.border}`
const th = { border, padding: '5px 8px', background: '#f1f5f9', fontWeight: 600, textAlign: 'center' as const, whiteSpace: 'nowrap' as const }
const td = { border, padding: '4px 8px', textAlign: 'right' as const, whiteSpace: 'nowrap' as const }
/** Nama di tiket bisa "PT. CPM" atau "CPM": urutan memakai kode tanpa awalan PT, judul kolom selalu "PT. KODE". */
const codeOf = (c: string) => c.replace(/^PT\.?\s*/i, '').trim()
const dash = (n: number) => (n ? formatNumber(n) : '-')

export default function RitaseByCustomerReport({ rows, totalLabel }: Props) {
  const found = [...new Set(rows.filter((r) => r.customer !== '').map((r) => r.customer))]
  const rank = (c: string) => { const i = CUSTOMERS.indexOf(codeOf(c)); return i < 0 ? CUSTOMERS.length : i }
  const customers = [...found].sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))

  const perUnit = new Map<string, Map<string, number>>()
  for (const r of rows) {
    const m = perUnit.get(r.equipment_code) ?? new Map<string, number>() // unit tanpa tiket (customer kosong) tetap tampil dengan 0
    if (r.customer !== '') m.set(r.customer, (m.get(r.customer) ?? 0) + r.count)
    perUnit.set(r.equipment_code, m)
  }
  const units = [...perUnit.keys()].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
  const grand = rows.reduce((s, r) => s + r.count, 0)
  const running = [...perUnit.values()].filter((m) => [...m.values()].some((n) => n > 0)).length // unit yang punya ritase
  const half = Math.ceil(units.length / 2)
  const blocks = units.length ? [units.slice(0, half), units.slice(half)].filter((b) => b.length) : []

  const block = (list: string[], offset: number) => (
    <div key={offset} style={{ background: '#fff', border, borderRadius: 12, overflowX: 'auto' }}>
      <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
        <thead>
          <tr>
            <th rowSpan={2} style={th}>NO</th>
            <th rowSpan={2} style={th}>CN UNIT</th>
            <th colSpan={Math.max(customers.length, 1)} style={th}>RITASE HAULING MITRA</th>
          </tr>
          <tr>
            {customers.length ? customers.map((c) => <th key={c} style={{ ...th, fontWeight: 500 }}>{c === '-' ? '-' : `PT. ${codeOf(c)}`}</th>) : <th style={{ ...th, fontWeight: 500 }}>-</th>}
          </tr>
        </thead>
        <tbody>
          {list.map((u, i) => (
            <tr key={u}>
              <td style={{ ...td, textAlign: 'center' }}>{offset + i + 1}</td>
              <td style={{ ...td, textAlign: 'left', fontWeight: 600 }}>{u}</td>
              {customers.length ? customers.map((c) => <td key={c} style={td}>{dash(perUnit.get(u)?.get(c) ?? 0)}</td>) : <td style={td} />}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {!units.length && <Alert type="info" showIcon title="Tidak ada unit di Equipment Activation pada tanggal ini" />}
      {!!units.length && <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, alignItems: 'start' }}>{blocks.map((b, i) => block(b, i * half))}</div>}
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 24, background: '#fff', border, borderRadius: 12, padding: '10px 16px', fontSize: 14 }}>
        <span>Total Ritase {totalLabel}: <b>{formatNumber(grand)}</b></span>
        <span>Total DT Run {totalLabel}: <b>{formatNumber(running)}</b></span>
      </div>
    </div>
  )
}
