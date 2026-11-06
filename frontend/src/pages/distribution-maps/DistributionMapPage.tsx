// Abnormal Distribution Map — sebaran alert di peta (3fr) + daftar kartu alert (1fr).
import { useMemo, useState } from 'react'
import { Empty } from 'antd'

import { useAlerts } from '@/hooks/useAlerts'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useLiveSocket } from '@/hooks/useLiveSocket'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { useZones } from '@/hooks/useZones'
import { alertTypeColor, alertTypeLabel } from '@/utils/colors'
import { formatDuration, minutesBetween } from '@/utils/format'
import SlotFilters, { SlotSelect } from '@/components/filters/SlotFilters'
import BaseMap, { FlyTo } from '@/components/map/BaseMap'
import { Dot, FitTo, Legend, ZoneLayer } from '@/components/map/layers'
import MapShell from '@/components/map/MapShell'
import InfoRows from '@/components/monitoring/InfoRows'
import ListHeader from '@/components/monitoring/ListHeader'
import styles from '@/components/monitoring/monitoring.module.css'
import PageHeader from '@/components/ui/PageHeader'

import AlertCard from './components/AlertCard'

const TYPE_OPTIONS = Object.entries(alertTypeLabel).map(([value, label]) => ({ value, label }))

export default function DistributionMapPage() {
  useLiveSocket() // alert baru → peta diperbarui
  const { data: zones } = useZones()
  const codeOf = useCodeOf()
  const { values, set, slot } = useSlotFilters({ type: undefined })
  const alerts = useAlerts({ date: slot.date, shift: slot.shift, equipmentId: slot.equipmentId, type: values.type })
  const [selected, setSelected] = useState<number>()
  const [fly, setFly] = useState(0)

  const rows = useMemo(
    () => (alerts.data?.data ?? []).filter((a) => a.latitude != null).map((a) => ({ ...a, code: codeOf(a.equipment_id) })),
    [alerts.data, codeOf],
  )
  const sel = rows.find((r) => r.id === selected)
  const pick = (id: number) => {
    setSelected(id)
    setFly((n) => n + 1)
  }

  const panel = (
    <>
      <SlotFilters values={values} onChange={set} extra={<SlotSelect label="Alert" value={values.type} options={TYPE_OPTIONS} onChange={(v) => set({ type: v })} />} />
      <ListHeader title="Abnormal Alert List" count={rows.length} />
      <div className={`${styles.list} ${styles.listBox}`}>
        {!rows.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={alerts.isFetching ? 'Memuat…' : 'Tidak ada alert'} />}
        {rows.map((a) => (
          <AlertCard key={a.id} alert={a} active={a.id === selected} onClick={() => pick(a.id)} />
        ))}
      </div>
    </>
  )

  return (
    <>
      <PageHeader title="Abnormal Distribution Map" />
      <MapShell
        panel={panel}
        map={(open) => (
          <BaseMap resizeKey={open}>
            <ZoneLayer zones={zones} />
            <FitTo points={rows.map((r) => [r.latitude!, r.longitude!])} area={zones} fitKey={`${slot.date}|${slot.shift}|${values.type}|${slot.equipmentId}|${rows.length > 0}`} />
            {rows.map((a) => (
              <Dot
                key={a.id}
                position={[a.latitude!, a.longitude!]}
                color={alertTypeColor[a.alert_type] ?? '#94a3b8'}
                selected={a.id === selected}
                onClick={() => setSelected(a.id)}
                popup={
                  <InfoRows
                    title={`${a.code} · ${alertTypeLabel[a.alert_type] ?? a.alert_type}`}
                    d={{
                      time: a.started_at, speed: a.speed_kph, fuelLiter: a.fuel_volume_liter, fuelPercent: a.fuel_percent, segment: a.segment_name,
                      vessel: a.vessel_status, lat: a.latitude, lon: a.longitude,
                      extra: [{ label: 'Durasi', value: a.ended_at ? formatDuration(minutesBetween(a.started_at, a.ended_at)) : 'masih berlangsung' }],
                    }}
                  />
                }
              />
            ))}
            <FlyTo position={sel ? [sel.latitude!, sel.longitude!] : null} trigger={fly} />
            <Legend title="Alert" items={TYPE_OPTIONS.map((t) => ({ color: alertTypeColor[t.value], label: t.label }))} />
          </BaseMap>
        )}
      />
    </>
  )
}
