// Report → Durasi Antrian Hauling. Filter: tanggal operasional, shift, unit. Isi laporan: components/report/HaulQueueReport.
import { Button } from 'antd'
import { Printer } from 'lucide-react'

import { useFilters } from '@/hooks/useFilters'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import { useShifts } from '@/hooks/useShifts'
import { appUrl } from '@/utils/basePath'
import { formatPlainDate } from '@/utils/format'
import { HAUL_END_ZONES, HAUL_START_SEGMENT } from '@/utils/haulCycle'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import HaulQueueReport from '@/components/report/HaulQueueReport'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportHaulQueuePage() {
  const now = useOperationalDefaults()
  const [f, set] = useFilters({ date: now.date })
  const { data: shifts = [] } = useShifts()
  const shiftName = f.shift ? (shifts.find((s) => s.shift_code === f.shift)?.shift_name ?? f.shift) : 'Semua shift'
  const printUrl = `/print/report-haul-queue?date=${f.date ?? ''}&shift=${f.shift ?? ''}&equipment=${f.equipment ?? ''}&preview=1`

  return (
    <>
      <PageHeader
        title="Durasi Antrian Hauling"
        subtitle={`${HAUL_START_SEGMENT} → ${HAUL_END_ZONES.join(' / ')}${f.date ? ` · tanggal operasional ${formatPlainDate(f.date)} · ${shiftName}` : ''}`}
        extra={
          <>
            <Button icon={<Printer size={14} />} disabled={!f.date} onClick={() => window.open(appUrl(printUrl), '_blank')}>Cetak / PDF</Button>
            <BackToReports />
          </>
        }
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={f} onChange={set} />
      </div>
      <HaulQueueReport date={f.date} shiftCode={f.shift} equipmentId={f.equipment} />
    </>
  )
}
