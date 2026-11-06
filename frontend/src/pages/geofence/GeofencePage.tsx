// Geofencing — peta kiri, panel 640 px: filter · ASSET PASSING + HOURLY PASSING SUMMARY · HOURLY TRAFFIC CHART.
import { useMemo } from 'react'
import { DatePicker, Select, Space } from 'antd'
import dayjs from 'dayjs'

import type { ZoneCollection } from '@/types/zone.types'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useCollapseSider } from '@/hooks/useCollapseSider'
import { useGeofenceEvents } from '@/hooks/useGeofenceEvents'
import { useLiveSocket } from '@/hooks/useLiveSocket'
import { useLiveStates } from '@/hooks/useLiveStates'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { useZones } from '@/hooks/useZones'
import { shiftHours, toLocal } from '@/utils/time'
import BaseMap from '@/components/map/BaseMap'
import { unitIcon } from '@/components/map/icons'
import { FitTo, UnitMarker, ZoneLayer } from '@/components/map/layers'
import MapShell from '@/components/map/MapShell'
import InfoRows from '@/components/monitoring/InfoRows'
import styles from '@/components/monitoring/monitoring.module.css'
import PageHeader from '@/components/ui/PageHeader'

import AssetPassing from './components/AssetPassing'
import HourlyPassingSummary from './components/HourlyPassingSummary'
import HourlyTrafficChart from './components/HourlyTrafficChart'

const ALL = '__all'
const PICK_AREA = 'Pilih area untuk menampilkan data'

export default function GeofencePage() {
  useCollapseSider()
  useLiveSocket() // kejadian baru → daftar diperbarui otomatis
  const { data: zones } = useZones()
  const { data: states = [] } = useLiveStates()
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  // tanggal & shift dikunci dan mengikuti shift yang sedang berjalan (berganti otomatis saat shift/tanggal berganti)
  const { values, set, slot } = useSlotFilters({ segment: undefined }, { follow: true })
  const segment = values.segment

  // Area pencatatan kejadian = area ORIG_FID 0 (aturan sistem lama, ADR 0009 C7)
  const areaNames = useMemo(
    () => [...new Set((zones?.features ?? []).filter((f) => f.properties.source_fid === 0).map((f) => f.properties.name))].sort(),
    [zones],
  )
  const events = useGeofenceEvents({ date: slot.date, shift: slot.shift }, segment)
  const rows = useMemo(() => (events.data?.data ?? []).map((e) => ({ ...e, code: codeOf(e.equipment_id) })), [events.data, codeOf])

  // jam-jam shift (melewati tengah malam bila perlu); tanpa shift = 24 jam mulai shift pertama
  const hours = useMemo(() => {
    const s = shifts.find((x) => x.shift_code === slot.shift)
    if (s) return shiftHours(s)
    const first = shifts[0]
    return first ? shiftHours({ start_time: first.start_time, end_time: first.start_time }) : Array.from({ length: 24 }, (_, i) => i)
  }, [shifts, slot.shift])
  const hourly = useMemo(
    () =>
      hours.map((h) => ({
        hour: `${String(h).padStart(2, '0')}:00`,
        in: rows.filter((r) => r.event_type === 'enter' && toLocal(r.event_time).hour() === h).length,
        out: rows.filter((r) => r.event_type === 'exit' && toLocal(r.event_time).hour() === h).length,
      })),
    [hours, rows],
  )
  // tabel hanya menampilkan jam yang ada kejadian; grafik tetap memakai semua jam
  const hourlyFilled = useMemo(() => hourly.filter((h) => h.in > 0 || h.out > 0), [hourly])
  const selectedArea = useMemo<ZoneCollection | undefined>(
    () => (zones && segment ? { ...zones, features: zones.features.filter((f) => f.properties.name === segment) } : undefined),
    [zones, segment],
  )

  const panel = (
    <>
      {/* sama dengan Live Tracking: label di kiri kolom isian, satu baris; bedanya di sini nilainya bisa diubah */}
      <div className={styles.panelFilters} style={{ gridTemplateColumns: '1fr 0.8fr 1.4fr' }}>
        <Space.Compact block>
          <Space.Addon>Tanggal</Space.Addon>
          <DatePicker
            style={{ width: '100%' }}
            format="DD/MM/YYYY"
            allowClear={false}
            disabled
            value={values.date ? dayjs(values.date) : null}
          />
        </Space.Compact>
        <Space.Compact block>
          <Space.Addon>Shift</Space.Addon>
          <Select
            style={{ width: '100%' }}
            disabled
            value={values.shift}
            options={shifts.map((x) => ({ value: x.shift_code, label: x.shift_name }))}
          />
        </Space.Compact>
        <Space.Compact block>
          <Space.Addon>Area</Space.Addon>
          <Select
            showSearch={{ optionFilterProp: 'label' }}
            style={{ width: '100%' }}
            value={segment ?? ALL}
            options={[{ value: ALL, label: 'Semua area' }, ...areaNames.map((n) => ({ value: n, label: n }))]}
            onChange={(v: string) => set({ segment: v === ALL ? undefined : v })}
          />
        </Space.Compact>
      </div>
      <div className={styles.geoBody}>
        <div className={styles.geoTop}>
          <AssetPassing rows={segment ? rows : []} emptyText={segment ? undefined : PICK_AREA} />
          <HourlyPassingSummary rows={segment ? hourlyFilled : []} emptyText={segment ? undefined : PICK_AREA} />
        </div>
        <HourlyTrafficChart hours={segment ? hourly : []} emptyText={segment ? undefined : PICK_AREA} />
      </div>
    </>
  )

  return (
    <>
      <PageHeader title="Geofencing" />
      <MapShell
        panelWidth="540px"
        panel={panel}
        map={(open) => (
          <BaseMap resizeKey={open}>
            <ZoneLayer zones={zones} highlight={segment} onSelect={(n) => areaNames.includes(n) && set({ segment: n })} />
            <FitTo area={selectedArea ?? zones} fitKey={segment ?? 'all'} />
            {states.filter((s) => s.latitude != null).map((s) => (
              <UnitMarker
                key={s.equipment_id}
                position={[s.latitude!, s.longitude!]}
                icon={unitIcon({ code: codeOf(s.equipment_id), status: s.equipment_status, vessel: s.vessel_status, heading: s.heading_deg, showLabel: true })}
                popup={<InfoRows title={codeOf(s.equipment_id)} d={{ time: s.event_time, speed: s.speed_kph, segment: s.segment_name, status: s.equipment_status, vessel: s.vessel_status, lat: s.latitude, lon: s.longitude }} />}
                onClick={() => s.segment_name && areaNames.includes(s.segment_name) && set({ segment: s.segment_name })}
              />
            ))}
          </BaseMap>
        )}
      />
    </>
  )
}
