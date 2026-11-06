// Report → Executive Summary Report Hauling. Tanggal laporan = H-1 dari tanggal operasional yang dipilih (awal: yang sedang berjalan).
import dayjs from 'dayjs'
import { Alert, Button, Spin } from 'antd'
import { Printer } from 'lucide-react'

import { errorMessage } from '@/services/http/ApiError'
import { useExecutiveHauling } from '@/hooks/useExecutiveHauling'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { formatPlainDate } from '@/utils/format'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import ExecutiveHaulingReport from '@/components/report/ExecutiveHaulingReport'
import { appUrl } from '@/utils/basePath'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportExecutiveHaulingPage() {
  const { values, set } = useSlotFilters()
  const reportDate = values.date ? dayjs(values.date).subtract(1, 'day').format('YYYY-MM-DD') : undefined
  const { data, isLoading, error } = useExecutiveHauling(reportDate)

  return (
    <>
      <PageHeader
        title="Executive Summary Report Hauling (Masih dalam tahap pengembangan)"
        subtitle={reportDate ? `Data sampai ${formatPlainDate(reportDate)} (H-1 dari tanggal operasional)` : undefined}
        extra={
          <>
            <Button icon={<Printer size={14} />} onClick={() => window.open(appUrl(`/print/report-executive-hauling?date=${values.date ?? ''}&preview=1`), '_blank')}>Cetak / PDF</Button>
            <BackToReports />
          </>
        }
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }]} values={values} onChange={set} />
      </div>
      {error && <Alert type="error" showIcon style={{ marginBottom: 12 }} title={errorMessage(error)} />}
      {isLoading ? <Spin style={{ display: 'block', margin: 60 }} /> : <ExecutiveHaulingReport rows={data ?? []} />}
    </>
  )
}
