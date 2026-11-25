// Isi laporan Durasi Antrian Hauling (dipakai halaman laporan dan halaman cetak). Data & aturan sama dengan Position History
// (utils/haulCycle.ts): dari unit berhenti di Km 01+000 - 02+000 sampai keluar dari Stockpile Emplacement / TLS2 Zona 4.
// Satu unit bisa punya 2–3 ritase dalam satu shift, jadi tabel dikelompokkan per unit (Asset ID) dengan ringkasan di kepala kelompoknya.
import { useEffect, useMemo } from 'react'
import { Alert, Empty, Progress, Tag } from 'antd'

import { useEquipmentList } from '@/hooks/useEquipments'
import { useHaulQueueReport } from '@/hooks/useHaulQueueReport'
import { useShifts } from '@/hooks/useShifts'
import { brand } from '@/utils/colors'
import { hasVessel } from '@/utils/equipmentType'
import { formatNumber, formatTime } from '@/utils/format'
import { formatSeconds } from '@/utils/haulCycle'

interface Props {
  date?: string
  shiftCode?: string
  /** id unit yang dipilih; kosong = semua Dump Truck aktif */
  equipmentId?: string
  /** dipanggil saat status memuat berubah (halaman cetak menunggu sampai selesai) */
  onLoadingChange?: (loading: boolean) => void
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

export default function HaulQueueReport({ date, shiftCode, equipmentId, onLoadingChange }: Props) {
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

  const groups = useMemo(() => units.map((u) => results.get(u.id)).filter((r) => !!r && r.cycles.length > 0), [units, results])
  const all = groups.flatMap((g) => g!.cycles)
  const sum = all.reduce((s, c) => s + c.seconds, 0)
  const longest = all.reduce((m, c) => Math.max(m, c.seconds), 0)
  const failed = [...results.values()].filter((r) => r.error).length
  const truncated = [...results.values()].some((r) => r.truncated)
  const shiftName = (code: string | null) => (code ? (shifts.find((s) => s.shift_code === code)?.shift_name ?? code) : '-')

  useEffect(() => onLoadingChange?.(loading), [loading, onLoadingChange])

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
              const cs = g!.cycles
              const total = cs.reduce((s, c) => s + c.seconds, 0)
              return (
                <tbody key={g!.equipment.id}>
                  <tr>
                    <td colSpan={6} style={{ ...td, background: '#e5e7eb', fontWeight: 700 }}>
                      <span style={{ marginRight: 16 }}>{g!.equipment.equipment_code}</span>
                      <span style={{ fontWeight: 500, color: '#475569' }}>
                        {cs.length} ritase · total {formatSeconds(total)} · rata-rata {formatSeconds(total / cs.length)} · terlama {formatSeconds(Math.max(...cs.map((c) => c.seconds)))}
                      </span>
                    </td>
                  </tr>
                  {cs.map((c, n) => (
                    <tr key={c.startIdx}>
                      <td style={{ ...td, color: '#64748b' }}>{g!.equipment.equipment_code}</td>
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
