// Report → Alert Summary.
import { Button } from 'antd'
import { Printer } from 'lucide-react'

import { useAlerts } from '@/hooks/useAlerts'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import FilterBar from '@/components/filters/FilterBar'
import AlertSummaryReport from '@/components/report/AlertSummaryReport'
import BackToReports from '@/components/report/BackToReports'
import { appUrl } from '@/utils/basePath'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportAlertSummaryPage() {
  const { values, set, slot } = useSlotFilters()
  const { data: shifts = [] } = useShifts()
  const alerts = useAlerts({ date: slot.date, shift: slot.shift })
  const print = () => window.open(appUrl(`/print/report-alert-summary?date=${slot.date ?? ''}${slot.shift ? `&shift=${slot.shift}` : ''}&preview=1`), '_blank')
  return (
    <>
      <PageHeader title="Alert Summary" extra={<><Button icon={<Printer size={14} />} onClick={print}>Cetak / PDF</Button><BackToReports /></>} />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }]} values={values} onChange={set} />
      </div>
      <AlertSummaryReport alerts={alerts.data?.data ?? []} shift={shifts.find((s) => s.shift_code === slot.shift)} shifts={shifts} />
    </>
  )
}
