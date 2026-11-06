// 5 laporan harian (Ritase, Payload, Produksi, Breakdown, PA) = SATU halaman + konfigurasi.
import { useMemo } from 'react'
import { Card, Empty } from 'antd'

import type { DailyMatrixConfig, DailyMatrixKind, MatrixRow } from '@/types/report.types'
import { useBreakdownDaily, useWeighbridgeDaily } from '@/hooks/useDailyReport'
import { useEquipmentList } from '@/hooks/useEquipments'
import { useFilters } from '@/hooks/useFilters'
import { useShifts } from '@/hooks/useShifts'
import { formatNumber } from '@/utils/format'
import {
  breakdownHours,
  breakdownStyle,
  byUnit,
  daysBetween,
  hoursAvailable,
  pa,
  paStyle,
  payload,
  production,
  productionStyle,
  ritase,
  ritaseStyle,
  totals,
} from '@/utils/reports'
import FilterBar from '@/components/filters/FilterBar'
import PageHeader from '@/components/ui/PageHeader'

import BackToReports from './BackToReports'
import ReportMatrix from './ReportMatrix'

const CONFIG: Record<DailyMatrixKind, DailyMatrixConfig> = {
  ritase: { title: 'Ritase Daily', source: 'weighbridge', uom: (s) => (s ? 'rit/shift' : 'rit/day'), value: ritase, digits: 0, total: { title: 'Total', fn: totals.sum }, style: () => ritaseStyle },
  payload: { title: 'Payload Daily', source: 'weighbridge', uom: () => 'ton/rit', value: payload, digits: 2, total: { title: 'Average', fn: totals.avgPositive }, style: () => undefined },
  production: { title: 'Production Daily', source: 'weighbridge', uom: () => 'ton', value: production, digits: 2, total: { title: 'Total', fn: totals.sum }, style: () => productionStyle },
  breakdown: { title: 'Breakdown Daily', source: 'breakdown', uom: () => 'hours', value: breakdownHours, digits: 2, total: { title: 'Total', fn: totals.sum }, style: (s) => breakdownStyle(hoursAvailable(s)) },
  pa: {
    title: 'Physical Availability (PA) Daily',
    source: 'breakdown',
    uom: () => '%',
    value: breakdownHours,
    digits: 0,
    total: { title: 'Average', fn: totals.avg },
    style: () => paStyle,
    derive: (perDay, days, shift) => Object.fromEntries(days.map((d) => [d, pa(perDay[d], hoursAvailable(shift))])),
  },
}

export default function DailyMatrix({ kind }: { kind: DailyMatrixKind }) {
  const c = CONFIG[kind]
  const { data: shifts = [] } = useShifts()
  const { data: equipment = [] } = useEquipmentList(true)
  // periode sengaja kosong: data baru dimuat setelah periode dipilih
  const [f, set] = useFilters()
  const picked = !!f.from && !!f.to

  const codes = f.equipment ? f.equipment.split(',').map((id) => equipment.find((e) => e.id === id)?.equipment_code).filter(Boolean) as string[] : undefined
  const period = { from: f.from, to: f.to, shiftId: shifts.find((s) => s.shift_code === f.shift)?.id, codes }
  const wb = useWeighbridgeDaily(period, c.source === 'weighbridge')
  const bd = useBreakdownDaily(period, c.source === 'breakdown')
  const q = c.source === 'weighbridge' ? wb : bd

  const days = useMemo(() => (f.from && f.to ? daysBetween(f.from, f.to) : []), [f.from, f.to])
  const rows = useMemo<MatrixRow[]>(() => {
    const units = byUnit(q.data ?? [], c.value)
    return [...units.entries()].map(([code, perDay]) => {
      const values = c.derive ? c.derive(perDay, days, f.shift) : perDay
      const list = days.map((d) => values[d] ?? 0)
      return { code, values, total: c.total.fn(list) }
    })
  }, [q.data, c, days, f.shift])

  return (
    <>
      <PageHeader
        title={c.title}
        subtitle="Per Asset ID per tanggal operasional"
        extra={<BackToReports />}
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'dateRange', maxDays: 366 }, { type: 'shift', allowAll: true }, { type: 'equipment', multiple: true }]} values={f} onChange={set} />
      </div>
      {!picked ? (
        <Card>
          <Empty description="Pilih periode tanggal untuk menampilkan laporan" />
        </Card>
      ) : (
      <ReportMatrix
        title={`${c.title} · ${days.length} hari`}
        days={days}
        rows={rows}
        uom={c.uom(f.shift)}
        digits={c.digits}
        format={(v) => (v == null || (v === 0 && kind !== 'pa') ? '' : formatNumber(v, c.digits))}
        cellStyle={c.style(f.shift)}
        totalTitle={c.total.title}
        loading={q.isFetching}
        exportName={`report_${kind}_${f.from}_${f.to}`}
      />
      )}
    </>
  )
}
