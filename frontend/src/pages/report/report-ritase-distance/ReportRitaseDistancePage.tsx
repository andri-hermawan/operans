// Report → Ritase Based On Distance. Filter: tanggal operasional, shift, unit. Isi laporan: components/report/DistanceRitaseReport.
import { useState } from 'react'
import { Button } from 'antd'
import { Download, Printer } from 'lucide-react'

import { useFilters } from '@/hooks/useFilters'
import { useOperationalDefaults } from '@/hooks/useOperationalDefaults'
import { useShifts } from '@/hooks/useShifts'
import { appUrl } from '@/utils/basePath'
import { formatPlainDate } from '@/utils/format'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import DistanceRitaseReport from '@/components/report/DistanceRitaseReport'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportRitaseDistancePage() {
  const now = useOperationalDefaults()
  const [f, set] = useFilters({ date: now.date })
  const { data: shifts = [] } = useShifts()
  const [exp, setExp] = useState<{ run: () => void; loading: boolean; disabled: boolean }>()
  const shiftName = f.shift ? (shifts.find((s) => s.shift_code === f.shift)?.shift_name ?? f.shift) : 'Semua shift'
  const printUrl = `/print/report-ritase-distance?date=${f.date ?? ''}&shift=${f.shift ?? ''}&equipment=${f.equipment ?? ''}&preview=1`

  return (
    <>
      <PageHeader
        title="Ritase Based On Distance"
        subtitle={`Siklus penuh: kembali ke segmen awal dengan status muatan yang sama${f.date ? ` · tanggal operasional ${formatPlainDate(f.date)} · ${shiftName}` : ''}`}
        extra={
          <>
            <BackToReports />
            <Button icon={<Download size={14} />} onClick={exp?.run} loading={exp?.loading} disabled={!exp || exp.disabled}>Export Excel</Button>
            <Button icon={<Printer size={14} />} disabled={!f.date} onClick={() => window.open(appUrl(printUrl), '_blank')}>Cetak / PDF</Button>
          </>
        }
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }, { type: 'equipment' }]} values={f} onChange={set} />
      </div>
      <DistanceRitaseReport date={f.date} shiftCode={f.shift} equipmentId={f.equipment} exportName={`ritase_based_on_distance_${f.date}_${f.shift ?? 'semua-shift'}`} onExportState={setExp} />
    </>
  )
}
