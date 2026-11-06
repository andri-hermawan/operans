// Report → Equipment Shift Performance (di FMS lama masih data contoh).
import { useMemo } from 'react'

import type { ShiftPerformanceRow } from '@/types/report.types'
import type { Column } from '@/types/table.types'
import { useAlerts } from '@/hooks/useAlerts'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useBreakdownDaily, useWeighbridgeDaily } from '@/hooks/useDailyReport'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { useActivity } from '@/hooks/useTelemetry'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'

export default function EquipmentShiftPerformancePage() {
  const { values, set, slot } = useSlotFilters()
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  const shiftId = shifts.find((s) => s.shift_code === slot.shift)?.id
  const activity = useActivity({ date: slot.date, shift: slot.shift })
  const wb = useWeighbridgeDaily({ from: slot.date, to: slot.date, shiftId })
  const bd = useBreakdownDaily({ from: slot.date, to: slot.date, shiftId })
  const alerts = useAlerts({ date: slot.date, shift: slot.shift })

  const rows = useMemo<ShiftPerformanceRow[]>(() => {
    const m = new Map<string, ShiftPerformanceRow>()
    const get = (code: string) => m.get(code) ?? m.set(code, { code, running: 0, idle: 0, avg: 0, max: 0, mileage: 0, fuelUsed: 0, burn: 0, ritase: 0, production: 0, breakdown: 0, alerts: 0 }).get(code)!
    for (const a of activity.data?.data ?? []) {
      Object.assign(get(codeOf(a.equipment_id)), { running: a.running_hours, idle: a.idle_hours, avg: a.avg_speed, max: a.max_speed, mileage: a.mileage_km, fuelUsed: a.fuel_decrease_liter, burn: a.fuel_burn_km_per_liter })
    }
    for (const c of wb.data ?? []) Object.assign(get(c.equipment_code), { ritase: c.count, production: Math.round(c.sum_kg / 10) / 100 })
    for (const c of bd.data ?? []) get(c.equipment_code).breakdown = Math.round(c.sum_hours * 100) / 100
    for (const a of alerts.data?.data ?? []) get(codeOf(a.equipment_id)).alerts++
    return [...m.values()].sort((a, b) => a.code.localeCompare(b.code))
  }, [activity.data, wb.data, bd.data, alerts.data, codeOf])

  const columns: Column<ShiftPerformanceRow>[] = [
    col.code('code', 'Asset ID', { fixed: 'left', mobile: 'title' }),
    col.number('running', 'Running (jam)', 2),
    col.number('idle', 'Idle (jam)', 2),
    col.number('breakdown', 'Breakdown (jam)', 2),
    col.number('avg', 'Avg Speed', 1),
    col.number('max', 'Max Speed', 0),
    col.number('mileage', 'Mileage (km)', 2),
    col.number('fuelUsed', 'Fuel Turun (L)', 2),
    col.number('burn', 'Burn Ratio (km/L)', 2),
    col.number('ritase', 'Ritase', 0),
    col.number('production', 'Produksi (ton)', 2),
    col.number('alerts', 'Alert', 0),
  ]
  return (
    <>
      <PageHeader title="Equipment Shift Performance" extra={<BackToReports />} />
      <DataTable<ShiftPerformanceRow>
        rows={rows}
        columns={columns}
        rowKey="code"
        loading={activity.isFetching || wb.isFetching || bd.isFetching}
        exportName={`shift_performance_${slot.date}_${slot.shift ?? 'all'}`}
        toolbar={<FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }]} values={values} onChange={set} />}
      />
    </>
  )
}
