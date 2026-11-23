// Report → Fuel Filling (raw data pengisian BBM per periode; filter sama dengan Fuel History).
import { useMemo } from 'react'

import type { FuelFilling } from '@/types/daily.types'
import type { Column } from '@/types/table.types'
import { useFilters } from '@/hooks/useFilters'
import { useFuelFillingHistory } from '@/hooks/useFuelFillingHistory'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'

const columns: Column<FuelFilling>[] = [
  col.plainDate('operational_date', 'Tanggal'),
  col.text('shift_name', 'Shift'),
  col.code('equipment_code', 'Asset ID', { mobile: 'title' }),
  col.text('equipment_model', 'Model'),
  col.number('hm_previous', 'HM Sebelumnya', 2),
  col.number('hm_filling', 'HM Pengisian', 2),
  col.number('total_hours', 'Total (jam)', 2),
  col.number('fuel_usage_liter', 'Pemakaian (L)', 2),
  col.number('fuel_consumption_lph', 'Konsumsi (L/jam)', 2),
  col.number('waste_removal_liter', 'Waste Removal (L)', 2),
  col.number('coal_mining_liter', 'Coal Mining (L)', 2),
  col.number('hauling_liter', 'Hauling (L)', 2),
  col.number('rental_liter', 'Rental (L)', 2),
  col.number('dewatering_liter', 'Dewatering (L)', 2),
  col.number('final_support_liter', 'Final Support (L)', 2),
]

export default function ReportFuelFillingPage() {
  const now = useOperationalDefaults()
  const [f, set] = useFilters({ from: now.date, to: now.date })
  const q = useFuelFillingHistory(f)
  // Terbaru di atas: tanggal operasional (desc), lalu HM pengisian (desc).
  const rows = useMemo(
    () => [...(q.data ?? [])].sort((a, b) => (a.operational_date < b.operational_date ? 1 : a.operational_date > b.operational_date ? -1 : (b.hm_filling ?? 0) - (a.hm_filling ?? 0) || b.id - a.id)),
    [q.data],
  )
  return (
    <>
      <PageHeader title="Fuel Filling" subtitle="Pengisian BBM unit per periode (tanggal, shift, asset)" extra={<BackToReports />} />
      <DataTable<FuelFilling>
        rows={rows}
        columns={columns}
        rowKey="id"
        loading={q.isFetching}
        error={q.error}
        onRetry={() => q.refetch()}
        exportName={`fuel_filling_${f.from}_${f.to}`}
        toolbar={<FilterBar fields={[{ type: 'dateRange', maxDays: 93 }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={f} onChange={set} />}
      />
    </>
  )
}
