// Halaman cetak Ritase Based On Distance (pratinjau ?preview=1 dari tombol "Cetak / PDF": ?date&shift&equipment).
import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { useCodeOf } from '@/hooks/useCodeOf'
import { useShifts } from '@/hooks/useShifts'
import { formatPlainDate } from '@/utils/format'
import DistanceRitaseReport from '@/components/report/DistanceRitaseReport'

import PrintFrame from './PrintFrame'

export default function PrintReportRitaseDistancePage() {
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
      title="Ritase Based On Distance"
      subtitle={(d) =>
        `Tanggal operasional ${formatPlainDate(d)} · ${shift ? `${shift.shift_name} (${shift.start_time}–${shift.end_time})` : 'Semua shift'}${equipmentId ? ` · ${codeOf(equipmentId)}` : ''}`
      }
      loading={false}
      ready={!loading}
    >
      {() => <DistanceRitaseReport date={date} shiftCode={shiftCode} equipmentId={equipmentId} onLoadingChange={setLoading} />}
    </PrintFrame>
  )
}
