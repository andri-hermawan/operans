// Tabel "OPERATION PERFORMANCE" (Executive Summary): PARAMETER · UOM · DAILY / MTD / YTD × PLAN · ACTUAL · VAR · ACH.
// Dipakai laporan Executive Summary Hauling dan Emplacement. Periode yang tidak punya nilai pada satu baris dikosongkan.
import type { Cell, Section } from '@/types/report.types'
import { brand } from '@/utils/colors'
import { PERIODS } from '@/utils/performance'

const num = (v: number) => Math.round(v).toLocaleString('en-US')
const ach = (c: Cell) => (c.plan > 0 ? Math.round((c.actual / c.plan) * 100) : 0)

const border = `1px solid ${brand.border}`
const th = { border, padding: '6px 10px', background: '#f1f5f9', fontWeight: 600, textAlign: 'center' as const }
const td = { border, padding: '6px 10px', textAlign: 'right' as const, whiteSpace: 'nowrap' as const }

export default function PerformanceTable({ sections }: { sections: Section[] }) {
  return (
    <div style={{ background: '#fff', border, borderRadius: 12, overflowX: 'auto' }}>
      <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
        <thead>
          <tr>
            <th colSpan={14} style={{ ...th, background: '#fff', fontSize: 15 }}>OPERATION PERFORMANCE</th>
          </tr>
          <tr>
            <th rowSpan={2} style={{ ...th, textAlign: 'left' }}>PARAMETER</th>
            <th rowSpan={2} style={th}>UOM</th>
            {PERIODS.map((p) => (
              <th key={p.key} colSpan={4} style={th}>{p.label}</th>
            ))}
          </tr>
          <tr>
            {PERIODS.flatMap((p) => ['PLAN', 'ACTUAL', 'VAR', 'ACH'].map((h) => <th key={`${p.key}-${h}`} style={{ ...th, fontWeight: 500 }}>{h}</th>))}
          </tr>
        </thead>
        <tbody>
          {sections.flatMap((s) => [
            <tr key={s.title}>
              <td colSpan={14} style={{ ...td, textAlign: 'left', background: '#e5e7eb', fontWeight: 700 }}>{s.title}</td>
            </tr>,
            ...s.lines.map((l) => (
              <tr key={`${s.title}-${l.label}`} style={l.bold ? { fontWeight: 600 } : undefined}>
                <td style={{ ...td, textAlign: 'left' }}>{l.label}</td>
                <td style={{ ...td, textAlign: 'center' }}>Ton</td>
                {PERIODS.flatMap((p) => {
                  const c = l.cells[p.key]
                  if (!c) return [0, 1, 2, 3].map((i) => <td key={`${p.key}-${i}`} style={td} />)
                  const a = ach(c)
                  return [
                    <td key={`${p.key}-p`} style={td}>{num(c.plan)}</td>,
                    <td key={`${p.key}-a`} style={td}>{num(c.actual)}</td>,
                    <td key={`${p.key}-v`} style={td}>{num(c.actual - c.plan)}</td>,
                    <td key={`${p.key}-h`} style={{ ...td, fontWeight: 700, color: a >= 100 ? '#1677ff' : brand.danger }}>{a}%</td>,
                  ]
                })}
              </tr>
            )),
          ])}
        </tbody>
      </table>
    </div>
  )
}
