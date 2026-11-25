// Halaman cetak Durasi Antrian Hauling (pratinjau ?preview=1 dari tombol "Cetak / PDF": ?date&shift&equipment).
import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { useCodeOf } from '@/hooks/useCodeOf'
import { useShifts } from '@/hooks/useShifts'
import { formatPlainDate } from '@/utils/format'
import { HAUL_END_ZONES, HAUL_START_SEGMENT } from '@/utils/haulCycle'
import HaulQueueReport from '@/components/report/HaulQueueReport'

import PrintFrame from './PrintFrame'

export default function PrintReportHaulQueuePage() {
  const [params] = useSearchParams()
  const date = params.get('date') ?? undefined
  const shiftCode = params.get('shift') || undefined
  const equipmentId = params.get('equipment') || undefined
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  const [loading, setLoading] = useState(true)
  const shift = shifts.find((s) => s.shift_code === shiftCode)
  return (
    <PrintFrame
      title="Durasi Antrian Hauling"
      subtitle={(d) =>
        `${HAUL_START_SEGMENT} → ${HAUL_END_ZONES.join(' / ')} · tanggal operasional ${formatPlainDate(d)} · ${shift ? `${shift.shift_name} (${shift.start_time}–${shift.end_time})` : 'Semua shift'}${equipmentId ? ` · ${codeOf(equipmentId)}` : ''}`
      }
      loading={false}
      ready={!loading}
    >
      {() => <HaulQueueReport date={date} shiftCode={shiftCode} equipmentId={equipmentId} onLoadingChange={setLoading} />}
    </PrintFrame>
  )
}
