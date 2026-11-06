// Halaman cetak Executive Summary Report Hauling (dibuka core-api untuk PDF terjadwal, atau pratinjau ?preview=1).
import { useSearchParams } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useExecutiveHauling } from '@/hooks/useExecutiveHauling'
import ExecutiveHaulingReport from '@/components/report/ExecutiveHaulingReport'

import { dayBefore, executiveSubtitle } from './executivePrint'
import PrintFrame from './PrintFrame'

export default function PrintReportExecutiveHaulingPage() {
  const date = useSearchParams()[0].get('date') ?? undefined
  const q = useExecutiveHauling(date ? dayBefore(date) : undefined)
  return (
    <PrintFrame title="Executive Summary Report Hauling" subtitle={executiveSubtitle} loading={q.isLoading} ready={q.isSuccess} error={q.error ? errorMessage(q.error) : undefined}>
      {() => <ExecutiveHaulingReport rows={q.data ?? []} />}
    </PrintFrame>
  )
}
