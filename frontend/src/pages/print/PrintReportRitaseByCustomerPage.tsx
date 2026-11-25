// Halaman cetak Ritase Hauling per Customer (dibuka core-api untuk PDF terjadwal: ?date&shift, atau pratinjau ?preview=1).
import { useSearchParams } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useRitaseByCustomer } from '@/hooks/useRitaseByCustomer'
import { useShifts } from '@/hooks/useShifts'
import { formatPlainDate } from '@/utils/format'
import RitaseByCustomerReport from '@/components/report/RitaseByCustomerReport'

import PrintFrame from './PrintFrame'

export default function PrintReportRitaseByCustomerPage() {
  const [params] = useSearchParams()
  const date = params.get('date') ?? undefined
  const shiftCode = params.get('shift') || undefined
  const shifts = useShifts()
  const q = useRitaseByCustomer(date, shiftCode)
  const shift = shifts.data?.find((s) => s.shift_code === shiftCode)
  const label = shift ? shift.shift_name : 'Semua Shift'
  return (
    <PrintFrame
      title="Ritase Hauling per Customer"
      subtitle={(d) => `Tiket timbangan · tanggal operasional ${formatPlainDate(d)} · ${shift ? `${shift.shift_name} (${shift.start_time}–${shift.end_time})` : 'Semua shift'}`}
      loading={q.isLoading || shifts.isLoading}
      ready={q.isSuccess}
      error={q.error ? errorMessage(q.error) : undefined}
    >
      {() => <RitaseByCustomerReport rows={q.data ?? []} totalLabel={label} />}
    </PrintFrame>
  )
}
