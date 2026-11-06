// Abnormal Alerts — daftar alert (kolom sama dengan FMS lama) + Export Excel.
// Halaman per jenis (Fuel/Overspeed/Underspeed/Off Track) = halaman ini dengan jenis terkunci.
import { useMemo } from 'react'
import { Tag } from 'antd'

import type { AlertListRow } from '@/types/alert.types'
import type { FilterField } from '@/types/filter.types'
import type { Column } from '@/types/table.types'
import { useAlerts } from '@/hooks/useAlerts'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useFilters } from '@/hooks/useFilters'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import { useShifts } from '@/hooks/useShifts'
import { alertTypeColor, alertTypeLabel, vesselStatusColor, vesselStatusLabel } from '@/utils/colors'
import { formatNumber, formatTime, minutesBetween } from '@/utils/format'
import FilterBar from '@/components/filters/FilterBar'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import AlertTypeTag from '@/components/ui/AlertTypeTag'
import PageHeader from '@/components/ui/PageHeader'
import StatusTag from '@/components/ui/StatusTag'

const TYPE_OPTIONS = Object.entries(alertTypeLabel).map(([value, label]) => ({ value, label }))

export default function AlertList({ types, title = 'Abnormal Alerts' }: { types?: string[]; title?: string }) {
  const now = useOperationalDefaults()
  const codeOf = useCodeOf()
  const { data: shifts = [] } = useShifts()
  const [f, set] = useFilters({ from: now.date, to: now.date })
  const typeParam = types?.join(',') ?? f.type
  const alerts = useAlerts({ date: f.from, dateTo: f.to, shift: f.shift, equipmentId: f.equipment, type: typeParam })

  const rows = useMemo<AlertListRow[]>(
    () =>
      (alerts.data?.data ?? []).map((a) => ({
        ...a,
        code: codeOf(a.equipment_id),
        duration: minutesBetween(a.started_at, a.ended_at),
        shift_name: shifts.find((s) => s.shift_code === a.shift_code)?.shift_name ?? a.shift_code ?? undefined,
      })),
    [alerts.data, codeOf, shifts],
  )

  const columns: Column<AlertListRow>[] = [
    col.rowNo(),
    col.tag('alert_type', 'Abnormal Alert', alertTypeColor, alertTypeLabel, { fixed: 'left', mobile: 'title', render: (r) => <AlertTypeTag type={r.alert_type} /> }),
    col.code('code', 'Asset ID', { fixed: 'left' }),
    col.plainDate('operational_date', 'Date'),
    col.text('shift_name', 'Shift'),
    col.time('started_at', 'Start Time'),
    { ...col.time<AlertListRow>('ended_at', 'Stop Time'), render: (r) => (r.ended_at ? formatTime(r.ended_at) : <Tag color="processing">berlangsung</Tag>) },
    col.number('duration', 'Duration (menit)', 2),
    col.number('speed_kph', 'Speed', 0),
    col.number('fuel_volume_liter', 'Fuel Volume (L)', 2),
    col.number('fuel_percent', 'Fuel %', 0),
    col.number('fuel_delta_liter', 'Fuel Diff (L)', 2),
    col.number('fuel_temperature_c', 'Fuel Temp', 1, { mobile: 'hide' }),
    col.text('segment_name', 'Map Segment'),
    col.coordinate(),
    col.tag('vessel_status', 'Vessel Status', vesselStatusColor, vesselStatusLabel),
    { key: 'is_engine_on', title: 'Engine', render: (r) => (r.is_engine_on == null ? '-' : <StatusTag value={r.is_engine_on ? 'on' : 'off'} colors={{ on: '#16a34a', off: '#64748b' }} labels={{ on: 'ON', off: 'OFF' }} />), value: (r) => (r.is_engine_on == null ? null : r.is_engine_on ? 'ON' : 'OFF') },
    col.number('threshold_value', 'Ambang', 2, { mobile: 'hide' }),
  ]

  const fields: FilterField[] = [
    { type: 'dateRange', maxDays: 93 },
    { type: 'shift', allowAll: true },
    { type: 'equipment' },
    ...(types ? [] : [{ type: 'select' as const, key: 'type', label: 'Alert', options: TYPE_OPTIONS, allowAll: true }]),
  ]

  return (
    <>
      <PageHeader title={title} subtitle={`${formatNumber(rows.length)} alert${alerts.data?.truncated ? ' (dipotong — persempit periode)' : ''}`} />
      <DataTable<AlertListRow>
        rows={rows}
        columns={columns}
        rowKey="id"
        loading={alerts.isFetching}
        error={alerts.error}
        onRetry={() => alerts.refetch()}
        searchable
        exportName={`alerts_${f.from}_${f.to}`}
        toolbar={<FilterBar fields={fields} values={f} onChange={set} />}
      />
    </>
  )
}
