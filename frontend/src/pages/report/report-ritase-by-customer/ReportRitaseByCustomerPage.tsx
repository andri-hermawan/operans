// Report → Ritase Hauling per Customer (dari tiket timbangan): harian, semua shift atau satu shift, dikelompokkan per customer.
import { Alert, Button, Spin } from 'antd'
import { Printer } from 'lucide-react'

import { errorMessage } from '@/services/http/ApiError'
import { useRitaseByCustomer } from '@/hooks/useRitaseByCustomer'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { appUrl } from '@/utils/basePath'
import { formatPlainDate } from '@/utils/format'
import FilterBar from '@/components/filters/FilterBar'
import BackToReports from '@/components/report/BackToReports'
import RitaseByCustomerReport from '@/components/report/RitaseByCustomerReport'
import PageHeader from '@/components/ui/PageHeader'

export default function ReportRitaseByCustomerPage() {
  const { values, set } = useSlotFilters()
  const shifts = useShifts()
  const { data, isLoading, error } = useRitaseByCustomer(values.date, values.shift)
  const shift = shifts.data?.find((s) => s.shift_code === values.shift)

  return (
    <>
      <PageHeader
        title="Ritase Hauling per Customer"
        subtitle={values.date ? `Tiket timbangan · ${formatPlainDate(values.date)} · ${shift ? shift.shift_name : 'Semua shift'}` : undefined}
        extra={
          <>
            <Button icon={<Printer size={14} />} onClick={() => window.open(appUrl(`/print/report-ritase-by-customer?date=${values.date ?? ''}&shift=${values.shift ?? ''}&preview=1`), '_blank')}>Cetak / PDF</Button>
            <BackToReports />
          </>
        }
      />
      <div style={{ marginBottom: 12 }}>
        <FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }]} values={values} onChange={set} />
      </div>
      {error && <Alert type="error" showIcon style={{ marginBottom: 12 }} title={errorMessage(error)} />}
      {isLoading ? <Spin style={{ display: 'block', margin: 60 }} /> : <RitaseByCustomerReport rows={data ?? []} totalLabel={shift ? shift.shift_name : 'Semua Shift'} />}
    </>
  )
}
