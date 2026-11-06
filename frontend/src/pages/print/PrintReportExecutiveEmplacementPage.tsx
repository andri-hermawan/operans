// Halaman cetak Executive Summary Report Emplacement (dibuka core-api untuk PDF terjadwal, atau pratinjau ?preview=1).
import { useSearchParams } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useExecutiveEmplacement } from '@/hooks/useExecutiveEmplacement'
import ExecutiveEmplacementReport from '@/components/report/ExecutiveEmplacementReport'

import { dayBefore, executiveSubtitle } from './executivePrint'
import PrintFrame from './PrintFrame'

export default function PrintReportExecutiveEmplacementPage() {
  const date = useSearchParams()[0].get('date') ?? undefined
  const q = useExecutiveEmplacement(date ? dayBefore(date) : undefined)
  return (
    <PrintFrame title="Executive Summary Report Emplacement" subtitle={executiveSubtitle} loading={q.isLoading} ready={q.isSuccess} error={q.error ? errorMessage(q.error) : undefined}>
      {() => <ExecutiveEmplacementReport rows={q.data ?? []} />}
    </PrintFrame>
  )
}
