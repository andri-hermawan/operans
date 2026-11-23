// Report → Breakdown History (raw data catatan breakdown per periode; filter sama dengan Fuel History).
import { useMemo } from 'react'
import { Tag } from 'antd'

import type { EquipmentBreakdown } from '@/types/daily.types'
import type { Column } from '@/types/table.types'
import { useBreakdownHistory } from '@/hooks/useBreakdownHistory'
import { useFilters } from '@/hooks/useFilters'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'

const STATUS: Record<string, { label: string; color: string }> = {
  breakdown: { label: 'Breakdown', color: 'error' },
  continue: { label: 'Continue', color: 'warning' },
  ready: { label: 'Ready', color: 'success' },
}

const columns: Column<EquipmentBreakdown>[] = [
  col.plainDate('operational_date', 'Tanggal'),
  col.text('shift_name', 'Shift'),
  col.code('equipment_code', 'Asset ID', { mobile: 'title' }),
  { key: 'status', title: 'Status', render: (r) => <Tag color={STATUS[r.status]?.color}>{STATUS[r.status]?.label ?? r.status}</Tag>, value: (r) => STATUS[r.status]?.label ?? r.status },
  col.text('category_label', 'Category'),
  col.dateTime('started_at', 'Mulai'),
  col.dateTime('ended_at', 'Selesai'),
  col.number('duration_hours', 'Durasi (jam)', 2),
  { key: 'repair_status_label', title: 'Repair Status', render: (r) => <Tag color={r.repair_status_code === 'done' ? 'success' : 'orange'}>{r.repair_status_label}</Tag>, value: (r) => r.repair_status_label },
  col.text('description', 'Keterangan'),
  col.text('location', 'Lokasi'),
]

export default function ReportBreakdownHistoryPage() {
  const now = useOperationalDefaults()
  const [f, set] = useFilters({ from: now.date, to: now.date })
  const q = useBreakdownHistory(f)
  // Terbaru di atas: waktu mulai (desc), yang belum ada jam mulai mengikuti tanggal operasionalnya.
  const rows = useMemo(() => {
    const t = (r: EquipmentBreakdown) => r.started_at ?? `${r.operational_date}T00:00:00Z`
    return [...(q.data ?? [])].sort((a, b) => (t(a) < t(b) ? 1 : t(a) > t(b) ? -1 : b.id - a.id))
  }, [q.data])
  return (
    <>
      <PageHeader title="Breakdown History" subtitle="Catatan breakdown per periode (tanggal, shift, asset)" extra={<BackToReports />} />
      <DataTable<EquipmentBreakdown>
        rows={rows}
        columns={columns}
        rowKey="id"
        loading={q.isFetching}
        error={q.error}
        onRetry={() => q.refetch()}
        exportName={`breakdown_history_${f.from}_${f.to}`}
        toolbar={<FilterBar fields={[{ type: 'dateRange', maxDays: 93 }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={f} onChange={set} />}
      />
    </>
  )
}
