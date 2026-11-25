// Isi laporan Durasi Antrian Hauling (dipakai halaman laporan dan halaman cetak). Data & aturan sama dengan Position History
// (utils/haulCycle.ts): dari unit berhenti di Km 01+000 - 02+000 sampai keluar dari Stockpile Emplacement / TLS2 Zona 4.
// Satu unit bisa punya 2–3 ritase dalam satu shift, jadi tabel dikelompokkan per unit (Asset ID) dengan ringkasan di kepala kelompoknya.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Empty, Progress, Tag } from 'antd'

import { useEquipmentList } from '@/hooks/useEquipments'
import { useHaulQueueReport } from '@/hooks/useHaulQueueReport'
import { useShifts } from '@/hooks/useShifts'
import { brand } from '@/utils/colors'
import { hasVessel } from '@/utils/equipmentType'
import { downloadExcel } from '@/utils/excel'
import { formatNumber, formatPlainDate, formatTime } from '@/utils/format'
import { formatSeconds } from '@/utils/haulCycle'

interface Props {
  date?: string
  shiftCode?: string
  /** id unit yang dipilih; kosong = semua Dump Truck aktif */
  equipmentId?: string
  /** dipanggil saat status memuat berubah (halaman cetak menunggu sampai selesai) */
  onLoadingChange?: (loading: boolean) => void
  /** nama berkas Excel (tanpa .xlsx); kosong di halaman cetak */
  exportName?: string
  /** tombol Export ada di header halaman: komponen ini melaporkan fungsi ekspor & statusnya ke halaman */
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

export default function HaulQueueReport({ date, shiftCode, equipmentId, onLoadingChange, exportName, onExportState }: Props) {
  const [exporting, setExporting] = useState(false)
  const { data: equipment = [] } = useEquipmentList(false)
  const { data: shifts = [] } = useShifts()

  // unit yang dihitung: Dump Truck (punya status muatan); bila Unit dipilih → hanya unit itu
  const units = useMemo(
    () =>
      equipment
        .filter((e) => (equipmentId ? e.id === equipmentId : hasVessel(e.equipment_type)))
        .sort((a, b) => a.equipment_code.localeCompare(b.equipment_code, undefined, { numeric: true })),
    [equipment, equipmentId],
  )
  const { results, done, total, loading } = useHaulQueueReport(date, shiftCode, units)

  // Urut dari "Mulai Antri" paling awal: ritase di dalam unit menurut waktu mulai, dan unit menurut ritase pertamanya
  // (waktu penuh, jadi shift malam yang melewati tengah malam tetap berurutan benar).
  const groups = useMemo(() => {
    const t = (iso: string) => new Date(iso).getTime()
    return units
      .map((u) => results.get(u.id))
      .filter((r) => !!r && r.cycles.length > 0)
      .map((r) => ({ ...r!, cycles: [...r!.cycles].sort((x, y) => t(x.start) - t(y.start)) }))
      .sort((x, y) => t(x.cycles[0].start) - t(y.cycles[0].start) || x.equipment.equipment_code.localeCompare(y.equipment.equipment_code, undefined, { numeric: true }))
  }, [units, results])
  const all = groups.flatMap((g) => g.cycles)
  const sum = all.reduce((s, c) => s + c.seconds, 0)
  const longest = all.reduce((m, c) => Math.max(m, c.seconds), 0)
  const failed = [...results.values()].filter((r) => r.error).length
  const truncated = [...results.values()].some((r) => r.truncated)
  const shiftName = (code: string | null) => (code ? (shifts.find((s) => s.shift_code === code)?.shift_name ?? code) : '-')

  useEffect(() => onLoadingChange?.(loading), [loading, onLoadingChange])

  // Excel: satu baris per ritase (urutan sama dengan tabel); durasi dalam menit (angka) agar bisa dijumlah/dirata-ratakan
  const doExport = async () => {
    if (!exportName) return
    setExporting(true)
    try {
      const rows = groups.flatMap((g) => g.cycles.map((c, n) => ({ code: g.equipment.equipment_code, n: n + 1, c })))
      await downloadExcel(exportName, [
        {
          name: 'Durasi Antrian',
          rows,
          columns: [
            { header: 'Tanggal Operasional', value: () => formatPlainDate(date) },
            { header: 'Asset ID', value: (r) => r.code },
            { header: 'Ritase', value: (r) => r.n },
            { header: 'Shift', value: (r) => shiftName(r.c.shiftCode) },
            { header: 'Mulai Antri', value: (r) => formatTime(r.c.start) },
            { header: 'Keluar Zona', value: (r) => formatTime(r.c.end) },
            { header: 'Durasi', value: (r) => formatSeconds(r.c.seconds) },
            { header: 'Durasi (menit)', numFmt: '0.0', value: (r) => Math.round((r.c.seconds / 60) * 10) / 10 },
            { header: 'Keterangan', value: (r) => (r.c.ongoing ? 'masih di zona' : '') },
          ],
        },
      ])
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
          <div style={{ fontSize: 13, marginBottom: 4 }}>Menghitung antrian per unit… {done} / {total} unit</div>
          <Progress percent={Math.round((done / total) * 100)} showInfo={false} size="small" />
        </div>
      )}
      {failed > 0 && <Alert type="warning" showIcon style={{ marginBottom: 12 }} title={`${failed} unit gagal dimuat datanya; ubah filter atau muat ulang halaman untuk mencoba lagi.`} />}
      {truncated && <Alert type="warning" showIcon style={{ marginBottom: 12 }} title="Data sebagian unit terpotong (terlalu banyak titik); durasi bisa kurang lengkap." />}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginBottom: 12 }}>
        <Stat label="Unit dengan antrian" value={formatNumber(groups.length)} hint={`dari ${formatNumber(total)} unit dicek`} />
        <Stat label="Total ritase" value={formatNumber(all.length)} />
        <Stat label="Rata-rata durasi" value={all.length ? formatSeconds(sum / all.length) : '-'} hint="per ritase" />
        <Stat label="Antrian terlama" value={all.length ? formatSeconds(longest) : '-'} />
      </div>

      {!groups.length ? (
        <div style={{ background: '#fff', border, borderRadius: 12, padding: 32 }}>
          <Empty description={loading ? 'Memuat data…' : 'Tidak ada antrian hauling pada tanggal ini'} />
        </div>
      ) : (
        <div style={{ background: '#fff', border, borderRadius: 12, overflowX: 'auto' }}>
          <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
            <thead>
              <tr>
                <th style={th}>Asset ID</th>
                <th style={{ ...th, textAlign: 'center' }}>Ritase</th>
                <th style={th}>Shift</th>
                <th style={th}>Mulai Antri</th>
                <th style={th}>Keluar Zona</th>
                <th style={{ ...th, textAlign: 'right' }}>Durasi</th>
              </tr>
            </thead>
            {groups.map((g) => {
              const cs = g.cycles
              const total = cs.reduce((s, c) => s + c.seconds, 0)
              return (
                <tbody key={g.equipment.id}>
                  <tr>
                    <td colSpan={6} style={{ ...td, background: '#e5e7eb', fontWeight: 700 }}>
                      <span style={{ marginRight: 16 }}>{g.equipment.equipment_code}</span>
                      <span style={{ fontWeight: 500, color: '#475569' }}>
                        {cs.length} ritase · total {formatSeconds(total)} · rata-rata {formatSeconds(total / cs.length)} · terlama {formatSeconds(Math.max(...cs.map((c) => c.seconds)))}
                      </span>
                    </td>
                  </tr>
                  {cs.map((c, n) => (
                    <tr key={c.startIdx}>
                      <td style={{ ...td, color: '#64748b' }}>{g.equipment.equipment_code}</td>
                      <td style={{ ...td, textAlign: 'center' }}>{n + 1}</td>
                      <td style={td}>{shiftName(c.shiftCode)}</td>
                      <td style={td}>{formatTime(c.start)}</td>
                      <td style={td}>
                        {formatTime(c.end)} {c.ongoing && <Tag color="warning" style={{ marginLeft: 6 }}>masih di zona</Tag>}
                      </td>
                      <td style={{ ...td, textAlign: 'right', fontWeight: 600 }}>{formatSeconds(c.seconds)}</td>
                    </tr>
                  ))}
                </tbody>
              )
            })}
          </table>
        </div>
      )}
    </>
  )
}
