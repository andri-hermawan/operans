// Report → Asset History (data mentah telemetri).
import { useMemo } from 'react'

import type { Column } from '@/types/table.types'
import type { PointRow } from '@/types/telemetry.types'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { usePoints } from '@/hooks/useTelemetry'
import { equipmentStatusColor, equipmentStatusLabel, vesselStatusColor, vesselStatusLabel } from '@/utils/colors'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'

const pointColumns: Column<PointRow>[] = [
  col.code('code', 'Asset ID', { fixed: 'left', mobile: 'title' }),
  col.plainDate('operational_date', 'Date'),
  col.time('event_time', 'Time'),
  col.text('shift_code', 'Shift'),
  col.engine<PointRow>(),
  col.number('speed_kph', 'Speed', 0),
  col.number('fuel_volume_liter', 'Fuel Volume (L)', 2),
  col.number('fuel_percent', 'Fuel %', 0),
  col.number('fuel_delta_liter', 'Fuel Diff (L)', 2),
  col.number('fuel_temperature_c', 'Fuel Temp', 1, { mobile: 'hide' }),
  col.text('segment_name', 'Map Segment'),
  col.coordinate(),
  col.tag('vessel_status', 'Vessel Status', vesselStatusColor, vesselStatusLabel),
  col.tag('equipment_status', 'Status', equipmentStatusColor, equipmentStatusLabel),
]

export default function ReportEquipmentLogsPage() {
  const { values, set, slot } = useSlotFilters()
  const codeOf = useCodeOf()
  const points = usePoints(slot, { stepSeconds: slot.equipmentId ? undefined : 30 })
  const rows = useMemo(() => (points.data?.data ?? []).map((p) => ({ ...p, code: codeOf(p.equipment_id) })), [points.data, codeOf])
  return (
    <>
      <PageHeader title="Asset History" subtitle={points.data?.truncated ? 'Data dipotong — pilih unit atau shift' : undefined} extra={<BackToReports />} />
      <DataTable<PointRow>
        rows={rows}
        columns={pointColumns}
        rowKey="id"
        loading={points.isFetching}
        error={points.error}
        onRetry={() => points.refetch()}
        exportName={`asset_history_${slot.date}`}
        toolbar={<FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={values} onChange={set} />}
      />
    </>
  )
}
