// Report → Executive Summary Report Emplacement. Tanggal laporan = H-1 dari tanggal operasional yang dipilih (awal: yang sedang berjalan).
import dayjs from 'dayjs'
import { Alert, Button, Spin } from 'antd'
import { Printer } from 'lucide-react'

import { errorMessage } from '@/services/http/ApiError'
import { useExecutiveEmplacement } from '@/hooks/useExecutiveEmplacement'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { formatPlainDate } from '@/utils/format'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import ExecutiveEmplacementReport from '@/components/report/ExecutiveEmplacementReport'
import { appUrl } from '@/utils/basePath'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportExecutiveEmplacementPage() {
  const { values, set } = useSlotFilters()
  const reportDate = values.date ? dayjs(values.date).subtract(1, 'day').format('YYYY-MM-DD') : undefined
  const { data, isLoading, error } = useExecutiveEmplacement(reportDate)

  return (
    <>
      <PageHeader
        title="Executive Summary Report Emplacement (Masih dalam tahap pengembangan)"
        subtitle={reportDate ? `Data sampai ${formatPlainDate(reportDate)} (H-1 dari tanggal operasional)` : undefined}
        extra={
          <>
            <Button icon={<Printer size={14} />} onClick={() => window.open(appUrl(`/print/report-executive-emplacement?date=${values.date ?? ''}&preview=1`), '_blank')}>Cetak / PDF</Button>
            <BackToReports />
          </>
        }
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }]} values={values} onChange={set} />
      </div>
      {error && <Alert type="error" showIcon style={{ marginBottom: 12 }} title={errorMessage(error)} />}
      {isLoading ? <Spin style={{ display: 'block', margin: 60 }} /> : <ExecutiveEmplacementReport rows={data ?? []} />}
    </>
  )
}
