// Report → Fuel History (kejadian isi/turun BBM).
import { useMemo } from 'react'
import { Tag } from 'antd'

import type { FuelEventRow } from '@/types/fuel.types'
import type { Column } from '@/types/table.types'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useFilters } from '@/hooks/useFilters'
import { useFuelEventHistory } from '@/hooks/useFuelEventHistory'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportFuelHistoryPage() {
  const now = useOperationalDefaults()
  const codeOf = useCodeOf()
  const [f, set] = useFilters({ from: now.date, to: now.date })
  const q = useFuelEventHistory(f)
  const rows = useMemo(() => (q.data ?? []).map((e) => ({ ...e, code: codeOf(e.equipment_id) })), [q.data, codeOf])
  const columns: Column<FuelEventRow>[] = [
    col.code('code', 'Asset ID', { mobile: 'title' }),
    col.dateTime('event_time', 'Waktu'),
    { key: 'event_type', title: 'Kejadian', render: (r) => <Tag color={r.event_type === 'increase' ? 'green' : 'red'}>{r.event_type === 'increase' ? 'Fuel Increase' : 'Fuel Decrease'}</Tag>, value: (r) => r.event_type },
    col.number('fuel_delta_liter', 'Selisih (L)', 2),
    col.number('fuel_volume_liter', 'Fuel Volume (L)', 2),
    col.text('description', 'Keterangan'),
    { key: 'alert_id', title: 'Alert', render: (r) => (r.alert_id ? <Tag color="red">Alert</Tag> : '-'), value: (r) => (r.alert_id ? 'Ya' : '') },
  ]
  return (
    <>
      <PageHeader title="Fuel History" subtitle="Kejadian isi & turun BBM yang berarti (penurunan per data ada di Asset History)" extra={<BackToReports />} />
      <DataTable<FuelEventRow>
        rows={rows}
        columns={columns}
        rowKey="id"
        loading={q.isFetching}
        error={q.error}
        onRetry={() => q.refetch()}
        exportName={`fuel_history_${f.from}_${f.to}`}
        toolbar={<FilterBar fields={[{ type: 'dateRange', maxDays: 93 }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={f} onChange={set} />}
      />
    </>
  )
}
