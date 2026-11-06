// Live Tracking — layout sama dengan FMS lama: peta kiri, panel 620 px kanan
// (filter · ASSET LIST + Activity Summary · ASSET STATUS · ABNORMAL ALERT SUMMARY).
import { useEffect, useMemo, useState } from 'react'
import { Input, Table, Tag, Tooltip } from 'antd'

import { useAlerts } from '@/hooks/useAlerts'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useCollapseSider } from '@/hooks/useCollapseSider'
import { useLiveSocket } from '@/hooks/useLiveSocket'
import { useLiveStates } from '@/hooks/useLiveStates'
import { useShifts } from '@/hooks/useShifts'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { useActivity } from '@/hooks/useTelemetry'
import { useZones } from '@/hooks/useZones'
import { formatPlainDate } from '@/utils/format'
import { withoutOutliers } from '@/utils/map'
import BaseMap, { FlyTo } from '@/components/map/BaseMap'
import { unitIcon } from '@/components/map/icons'
import { FitTo, GoHome, UnitMarker, ZoneLayer } from '@/components/map/layers'
import MapShell from '@/components/map/MapShell'
import AssetList from '@/components/monitoring/AssetList'
import AssetStatus from '@/components/monitoring/AssetStatus'
import InfoRows from '@/components/monitoring/InfoRows'
import styles from '@/components/monitoring/monitoring.module.css'
import AlertTypeTag from '@/components/ui/AlertTypeTag'
import PageHeader from '@/components/ui/PageHeader'
import SectionCard from '@/components/ui/SectionCard'

const LIVE_TEXT = { connecting: 'Menyambung…', live: 'Live', offline: 'Terputus — menyambung ulang' } as const

export default function TrackingPage() {
  useCollapseSider()
  const status = useLiveSocket()
  const { data: states = [] } = useLiveStates()
  const { data: zones } = useZones()
  const { data: shifts = [] } = useShifts()
  const codeOf = useCodeOf()
  // tanggal & shift dikunci dan mengikuti shift yang sedang berjalan (berganti otomatis saat shift/tanggal berganti)
  const { values, slot } = useSlotFilters({}, { follow: true })
  const [selected, setSelected] = useState<string>()
  const [search, setSearch] = useState('')
  const [fly, setFly] = useState(0)
  const [home, setHome] = useState(0)

  // Ringkasan aktivitas hanya untuk unit yang dibuka di Asset List (murah), dan dihitung ulang tiap data live unit itu berubah
  const activity = useActivity({ date: slot.date, shift: slot.shift, equipmentId: selected }, !!selected)
  const alerts = useAlerts({ date: slot.date, shift: slot.shift })
  const actOf = useMemo(() => new Map((activity.data?.data ?? []).map((a) => [a.equipment_id, a])), [activity.data])

  const units = useMemo(
    () => states.map((s) => ({ ...s, code: codeOf(s.equipment_id) })).filter((s) => s.code.toLowerCase().includes(search.toLowerCase())).sort((a, b) => a.code.localeCompare(b.code)),
    [states, codeOf, search],
  )
  const withPos = units.filter((u) => u.latitude != null && u.longitude != null)
  // Tampilan awal: unit AKTIF saja (yang offline posisinya sudah lama), tanpa pencilan, dan tidak terpengaruh kolom cari.
  // Bila belum ada unit aktif → seluruh area tambang.
  const fitPoints = useMemo(
    () => withoutOutliers(states.filter((s) => s.latitude != null && s.longitude != null && s.equipment_status && s.equipment_status !== 'offline').map((s) => [s.latitude!, s.longitude!] as [number, number])),
    [states],
  )
  const sel = units.find((u) => u.equipment_id === selected)
  const refetchActivity = activity.refetch
  const liveAt = sel?.event_time
  useEffect(() => {
    if (!selected) return
    const t = setTimeout(() => void refetchActivity(), 1500) // jeda singkat: beberapa data beruntun dihitung sekali
    return () => clearTimeout(t)
  }, [selected, liveAt, refetchActivity])
  // batal pilih (tombol X / klik unit yang sama) → peta kembali ke tampilan awal
  const clear = () => {
    setSelected(undefined)
    setHome((n) => n + 1)
  }
  const choose = (id: string) => {
    if (id === selected) return clear()
    setSelected(id)
    setFly((n) => n + 1)
  }

  const alertSummary = useMemo(() => {
    const m = new Map<string, { type: string; code: string; count: number }>()
    for (const a of alerts.data?.data ?? []) {
      const code = codeOf(a.equipment_id)
      const k = `${a.alert_type}|${code}`
      m.set(k, { type: a.alert_type, code, count: (m.get(k)?.count ?? 0) + 1 })
    }
    return [...m.values()].sort((a, b) => b.count - a.count)
  }, [alerts.data, codeOf])

  const shiftName = shifts.find((s) => s.shift_code === values.shift)?.shift_name ?? '-'

  const panel = (
    <>
      <div className={styles.panelFilters}>
        <Input.Search allowClear placeholder="Cari unit…" value={search} onChange={(e) => setSearch(e.target.value)} />
        <Input value={formatPlainDate(values.date)} disabled addonBefore="Tanggal" />
        <Input value={shiftName} disabled addonBefore="Shift" />
      </div>
      <div className={styles.fillGrid}>
        <SectionCard title={`Asset List`} className={styles.fillCard} styles={{ body: { padding: 0 } }}>
          <AssetList
            units={units}
            selectedId={selected}
            onSelect={choose}
            onClear={clear}
            activity={actOf}
            loading={activity.isLoading}
            onRefresh={() => activity.refetch()}
          />
        </SectionCard>
        <div className={styles.fillSplit}>
          <SectionCard title="Asset Status" className={styles.fillCard} styles={{ body: { padding: 0 } }}>
            <AssetStatus units={sel ? [sel] : units} />
          </SectionCard>
          <SectionCard title="Abnormal Alert Summary" className={styles.fillCard} styles={{ body: { padding: 0 } }}>
            <Table
              size="small"
              pagination={false}
              rowKey={(r) => `${r.type}|${r.code}`}
              dataSource={alertSummary}
              locale={{ emptyText: 'Tidak ada alert shift ini' }}
              columns={[
                { title: 'Event', dataIndex: 'type', render: (t: string) => <AlertTypeTag type={t} /> },
                { title: 'Asset ID', dataIndex: 'code' },
                { title: 'Count', dataIndex: 'count', align: 'right' },
              ]}
            />
          </SectionCard>
        </div>
      </div>
    </>
  )

  return (
    <>
      <PageHeader
        title="Live Tracking"
        extra={<Tooltip title="Koneksi data langsung"><Tag color={status === 'live' ? 'success' : status === 'connecting' ? 'processing' : 'error'}>{LIVE_TEXT[status]}</Tag></Tooltip>}
      />
      <MapShell
        panelWidth="620px"
        panel={panel}
        map={(open) => (
          <BaseMap resizeKey={open}>
            <ZoneLayer zones={zones} />
            <FitTo points={fitPoints} area={zones} fitKey={fitPoints.length ? 'units' : 'area'} />
            {withPos.map((u) => (
              <UnitMarker
                key={u.equipment_id}
                position={[u.latitude!, u.longitude!]}
                icon={unitIcon({ code: u.code, status: u.equipment_status, vessel: u.vessel_status, heading: u.heading_deg, alert: u.alert_count > 0, breakdown: !!u.is_breakdown, selected: u.equipment_id === selected, showLabel: true })}
                onClick={() => setSelected(u.equipment_id)}
                open={u.equipment_id === selected}
                popup={
                  <InfoRows
                    title={u.code}
                    d={{
                      time: u.event_time, speed: u.speed_kph, fuelLiter: u.fuel_volume_liter, fuelPercent: u.fuel_percent, segment: u.segment_name,
                      vessel: u.vessel_status, engine: u.is_engine_on, operator: u.operator_name, lat: u.latitude, lon: u.longitude,
                      extra: [
                        ...(u.alert_count ? [{ label: 'Alert shift', value: u.alert_count }] : []),
                        // deskripsi (mis. "AC HOT") hanya selama masih breakdown; sesudah ready baris ini hilang
                        ...(u.is_breakdown && u.breakdown_description ? [{ label: 'Breakdown', value: u.breakdown_description }] : []),
                      ],
                    }}
                  />
                }
              />
            ))}
            <GoHome trigger={home} />
            <FlyTo position={sel?.latitude != null ? [sel.latitude, sel.longitude!] : null} zoom={18} trigger={fly} />
          </BaseMap>
        )}
      />
    </>
  )
}
