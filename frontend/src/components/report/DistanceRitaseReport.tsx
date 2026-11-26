// Isi laporan Ritase Based On Distance (dipakai halaman laporan dan halaman cetak). Data sama dengan Position History;
// satu ritase = siklus penuh, kembali ke segmen awal dengan status muatan yang sama (utils/distanceRitase.ts).
// Tabel dikelompokkan per unit: baris pertama tiap unit = Total ritase, lalu rincian per ritase.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Empty, Progress, Tag } from 'antd'

import { useEquipmentList } from '@/hooks/useEquipments'
import { useDistanceRitaseReport, type UnitRitase } from '@/hooks/useDistanceRitaseReport'
import { brand } from '@/utils/colors'
import type { ExcelSheet } from '@/types/excel.types'
import type { DistanceRitase } from '@/utils/distanceRitase'
import { hasVessel } from '@/utils/equipmentType'
import { downloadExcel } from '@/utils/excel'
import { formatNumber, formatPlainDate, formatTime } from '@/utils/format'
import { formatSeconds } from '@/utils/haulCycle'

type DetailRow = { code: string; r: DistanceRitase }
type SummaryRow = { code: string; operator: string; count: number; seconds: number }

interface Props {
  date?: string
  shiftCode?: string
  /** id unit yang dipilih; kosong = semua Dump Truck */
  equipmentId?: string
  onLoadingChange?: (loading: boolean) => void
  /** nama berkas Excel (tanpa .xlsx); kosong di halaman cetak */
  exportName?: string
  onExportState?: (s: { run: () => void; loading: boolean; disabled: boolean }) => void
}

const border = `1px solid ${brand.border}`
const th = { border, padding: '8px 12px', background: '#f1f5f9', fontWeight: 600, textAlign: 'left' as const, whiteSpace: 'nowrap' as const }
const td = { border, padding: '6px 12px', whiteSpace: 'nowrap' as const }

const Stat = ({ label, value, hint }: { label: string; value: string; hint?: string }) => (
  <div style={{ flex: '1 1 180px', background: '#fff', border, borderRadius: 12, padding: '12px 16px' }}>
    <div style={{ fontSize: 12, color: '#64748b' }}>{label}</div>
    <div style={{ fontSize: 22, fontWeight: 700, lineHeight: 1.2 }}>{value}</div>
    {hint && <div style={{ fontSize: 12, color: '#64748b' }}>{hint}</div>}
  </div>
)

const operatorOf = (g: UnitRitase) => g.ritases.find((r) => r.operator)?.operator ?? ''

export default function DistanceRitaseReport({ date, shiftCode, equipmentId, onLoadingChange, exportName, onExportState }: Props) {
  const [exporting, setExporting] = useState(false)
  const { data: equipment = [] } = useEquipmentList(false)

  const units = useMemo(
    () =>
      equipment
        .filter((e) => (equipmentId ? e.id === equipmentId : hasVessel(e.equipment_type)))
        .sort((a, b) => a.equipment_code.localeCompare(b.equipment_code, undefined, { numeric: true })),
    [equipment, equipmentId],
  )
  const { results, done, total, loading } = useDistanceRitaseReport(date, shiftCode, units)

  // Urut dari jam Start paling awal (waktu penuh, agar shift malam yang melewati tengah malam tetap berurutan benar)
  const groups = useMemo(() => {
    const t = (iso: string) => new Date(iso).getTime()
    return units
      .map((u) => results.get(u.id))
      .filter((r): r is UnitRitase => !!r && r.ritases.length > 0)
      .map((r) => ({ ...r, ritases: [...r.ritases].sort((x, y) => t(x.start) - t(y.start)) }))
      .sort((x, y) => t(x.ritases[0].start) - t(y.ritases[0].start) || x.equipment.equipment_code.localeCompare(y.equipment.equipment_code, undefined, { numeric: true }))
  }, [units, results])
  const all = groups.flatMap((g) => g.ritases)
  const sum = all.reduce((s, r) => s + r.seconds, 0)
  const failed = [...results.values()].filter((r) => r.error).length
  const truncated = [...results.values()].some((r) => r.truncated)

  useEffect(() => onLoadingChange?.(loading), [loading, onLoadingChange])

  const doExport = async () => {
    if (!exportName) return
    setExporting(true)
    try {
      const rows: DetailRow[] = groups.flatMap((g) => g.ritases.map((r) => ({ code: g.equipment.equipment_code, r })))
      const detail: ExcelSheet<DetailRow> = {
        name: 'Ritase',
        rows,
        columns: [
          { header: 'Tanggal Operasional', value: () => formatPlainDate(date) },
          { header: 'Asset ID', value: (x) => x.code },
          { header: 'Operator', value: (x) => x.r.operator ?? '' },
          { header: 'Segment', value: (x) => x.r.segment },
          { header: 'Start', value: (x) => formatTime(x.r.start) },
          { header: 'End', value: (x) => formatTime(x.r.end) },
          { header: 'Duration', value: (x) => formatSeconds(x.r.seconds) },
          { header: 'Duration (menit)', numFmt: '0.0', value: (x) => Math.round((x.r.seconds / 60) * 10) / 10 },
          { header: 'Jarak (km)', numFmt: '0.0', value: (x) => Math.round(x.r.km * 10) / 10 },
          { header: 'Vessel Status', value: (x) => x.r.status },
          { header: 'Ritase', value: () => 1 },
        ],
      }
      const summary: ExcelSheet<SummaryRow> = {
        name: 'Total per Unit',
        rows: groups.map((g) => ({ code: g.equipment.equipment_code, operator: operatorOf(g), count: g.ritases.length, seconds: g.ritases.reduce((s, r) => s + r.seconds, 0) })),
        columns: [
          { header: 'Asset ID', value: (x) => x.code },
          { header: 'Operator', value: (x) => x.operator },
          { header: 'Total Ritase', value: (x) => x.count },
          { header: 'Total Durasi (menit)', numFmt: '0.0', value: (x) => Math.round((x.seconds / 60) * 10) / 10 },
        ],
      }
      await downloadExcel(exportName, [detail as ExcelSheet<unknown>, summary as ExcelSheet<unknown>])
    } finally {
      setExporting(false)
    }
  }

  const runRef = useRef(doExport)
  useEffect(() => {
    runRef.current = doExport
  })
  const run = useCallback(() => void runRef.current(), [])
  const empty = !groups.length
  useEffect(() => onExportState?.({ run, loading: exporting, disabled: empty || !exportName }), [onExportState, run, exporting, empty, exportName])

  return (
    <>
      {loading && total > 0 && (
        <div style={{ marginBottom: 12, background: '#fff', border, borderRadius: 12, padding: '10px 16px' }}>
          <div style={{ fontSize: 13, marginBottom: 4 }}>Menghitung ritase per unit… {done} / {total} unit</div>
          <Progress percent={Math.round((done / total) * 100)} showInfo={false} size="small" />
        </div>
      )}
      {failed > 0 && <Alert type="warning" showIcon style={{ marginBottom: 12 }} title={`${failed} unit gagal dimuat datanya; ubah filter atau muat ulang halaman untuk mencoba lagi.`} />}
      {truncated && <Alert type="warning" showIcon style={{ marginBottom: 12 }} title="Data sebagian unit terpotong (terlalu banyak titik); ritase bisa kurang lengkap." />}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 12 }}>
        <Stat label="Unit dengan ritase" value={formatNumber(groups.length)} hint={`dari ${formatNumber(total)} unit dicek`} />
        <Stat label="Total ritase" value={formatNumber(all.length)} />
        <Stat label="Rata-rata durasi" value={all.length ? formatSeconds(sum / all.length) : '-'} hint="per ritase" />
      </div>

      {!groups.length ? (
        <div style={{ background: '#fff', border, borderRadius: 12, padding: 32 }}>
          <Empty description={loading ? 'Memuat data…' : 'Tidak ada ritase lengkap pada tanggal ini'} />
        </div>
      ) : (
        <div style={{ background: '#fff', border, borderRadius: 12, overflowX: 'auto' }}>
          <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
            <thead>
              <tr>
                <th style={th}>Asset ID</th>
                <th style={th}>Operator</th>
                <th style={th}>Segment</th>
                <th style={th}>Start</th>
                <th style={th}>End</th>
                <th style={{ ...th, textAlign: 'right' }}>Duration</th>
                <th style={{ ...th, textAlign: 'right' }}>Jarak</th>
                <th style={th}>Vessel Status</th>
                <th style={{ ...th, textAlign: 'center' }}>Ritase</th>
              </tr>
            </thead>
            {groups.map((g) => (
              <tbody key={g.equipment.id}>
                <tr style={{ background: '#e5e7eb', fontWeight: 700 }}>
                  <td style={td}>{g.equipment.equipment_code}</td>
                  <td style={td} colSpan={6} />
                  <td style={td}>Total</td>
                  <td style={{ ...td, textAlign: 'center' }}>{g.ritases.length}</td>
                </tr>
                {g.ritases.map((r) => (
                  <tr key={r.startIdx}>
                    <td style={{ ...td, color: '#64748b' }}>{g.equipment.equipment_code}</td>
                    <td style={td}>{r.operator ?? '-'}</td>
                    <td style={td}>{r.segment}</td>
                    <td style={td}>{formatTime(r.start)}</td>
                    <td style={td}>{formatTime(r.end)}</td>
                    <td style={{ ...td, textAlign: 'right', fontWeight: 600 }}>{formatSeconds(r.seconds)}</td>
                    <td style={{ ...td, textAlign: 'right' }}>{formatNumber(r.km, 1)} km</td>
                    <td style={td}><Tag color={r.status === 'loaded' ? 'orange' : 'blue'} style={{ marginInlineEnd: 0 }}>{r.status}</Tag></td>
                    <td style={{ ...td, textAlign: 'center' }}>1</td>
                  </tr>
                ))}
              </tbody>
            ))}
          </table>
        </div>
      )}
    </>
  )
}
