import { useQuery } from '@tanstack/react-query'

import { getRitaseByCustomer } from '@/services/api/report.api'

import { useProjectKey } from './useProjectKey'
import { useShifts } from './useShifts'

/** Ritase (tiket timbangan) per unit per customer untuk satu tanggal operasional; shift kosong = semua shift. */
export const useRitaseByCustomer = (date?: string, shiftCode?: string) => {
  const { pid, key } = useProjectKey('report', 'ritase-by-customer', date, shiftCode)
  const shifts = useShifts()
  const shiftId = shiftCode ? shifts.data?.find((s) => s.shift_code === shiftCode)?.id : undefined
  return useQuery({
    queryKey: [...key, shiftId],
    enabled: !!pid && !!date && (!shiftCode || !!shiftId),
    queryFn: () => getRitaseByCustomer(pid!, date!, shiftId),
  })
}
