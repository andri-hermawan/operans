// Equipment Graphic — daftar unit + availability (dari Status Breakdown) dan grafik speed & fuel unit terpilih.
// Di FMS lama halaman ini masih data contoh; di sini memakai data sebenarnya.
import { useMemo } from 'react'
import { Empty, Tag } from 'antd'

import type { EquipmentAvailabilityRow } from '@/types/equipment.types'
import type { Column } from '@/types/table.types'
import { useEquipmentBreakdowns } from '@/hooks/useEquipmentBreakdowns'
import { useEquipmentList } from '@/hooks/useEquipments'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { usePoints } from '@/hooks/useTelemetry'
import { brand } from '@/utils/colors'
import { toLocal } from '@/utils/time'
import Chart from '@/components/chart/Chart'
import FilterBar from '@/components/filters/FilterBar'
import { col } from '@/components/table/columns'
import DataTable from '@/components/table/DataTable'
import PageHeader from '@/components/ui/PageHeader'
import SectionCard from '@/components/ui/SectionCard'

const AV_COLOR = { Ready: 'success', Breakdown: 'error', Continue: 'warning' } as const

export default function GraphicPage() {
  const { values, set, slot } = useSlotFilters({ equipment: undefined })
  const { data: equipment = [], isLoading } = useEquipmentList(false)
  const { data: shifts = [] } = useShifts()
  const shiftId = shifts.find((s) => s.shift_code === slot.shift)?.id
  const breakdowns = useEquipmentBreakdowns(slot.date, shiftId)
  const rows = useMemo<EquipmentAvailabilityRow[]>(() => {
    // status terakhir per unit pada tanggal itu
    const last = new Map<string, string>()
    for (const b of [...(breakdowns.data ?? [])].sort((a, z) => (a.started_at ?? '').localeCompare(z.started_at ?? ''))) last.set(b.equipment_code, b.status)
    return equipment.map((e) => {
      const st = last.get(e.equipment_code)
      return { ...e, availability: st === 'breakdown' ? 'Breakdown' : st === 'continue' ? 'Continue' : 'Ready' }
    })
  }, [equipment, breakdowns.data])

  const points = usePoints(slot, { enabled: !!slot.equipmentId })
  const data = points.data?.data ?? []
  const selected = equipment.find((e) => e.id === slot.equipmentId)

  const columns: Column<EquipmentAvailabilityRow>[] = [
    col.code('equipment_code', 'Equipment Code', { mobile: 'title' }),
    col.text('equipment_type', 'Type'),
    col.text('brand', 'Brand'),
    col.text('model', 'Model'),
    col.text('class', 'Class'),
    { key: 'availability', title: 'Availability', render: (r) => <Tag color={AV_COLOR[r.availability]}>{r.availability}</Tag>, value: (r) => r.availability },
  ]

  return (
    <>
      <PageHeader title="Equipment Graphic" subtitle="Klik unit untuk melihat grafik speed & fuel" />
      <DataTable<EquipmentAvailabilityRow>
        rows={rows}
        columns={columns}
        rowKey="id"
        loading={isLoading || breakdowns.isFetching}
        onRowClick={(r) => set({ equipment: r.id })}
        exportName={`equipment_availability_${slot.date}`}
        toolbar={<FilterBar fields={[{ type: 'date' }, { type: 'shift', allowAll: true }]} values={values} onChange={set} />}
      />
      <div style={{ marginTop: 12 }}>
        <SectionCard title={selected ? `Speed & Fuel % — ${selected.equipment_code}` : 'Speed & Fuel %'}>
          {!selected ? <Empty description="Pilih unit di tabel" /> : (
            <Chart
              height={300}
              loading={points.isFetching}
              option={{
                xAxis: { type: 'category', data: data.map((p) => toLocal(p.event_time).format('HH:mm')) },
                yAxis: [{ type: 'value', name: 'km/h' }, { type: 'value', name: '%', max: 100 }],
                series: [
                  { name: 'Speed (km/h)', type: 'line', showSymbol: false, areaStyle: { opacity: 0.15 }, data: data.map((p) => p.speed_kph), itemStyle: { color: brand.primary } },
                  { name: 'Fuel %', type: 'line', yAxisIndex: 1, showSymbol: false, data: data.map((p) => p.fuel_percent), itemStyle: { color: brand.accent } },
                ],
              }}
            />
          )}
        </SectionCard>
      </div>
    </>
  )
}
