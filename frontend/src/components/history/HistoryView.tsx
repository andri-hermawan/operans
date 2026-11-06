// Position History & Fuel History = SATU halaman dengan dua mode (di FMS lama 95% salinan).
// Peta + playback (kiri atas), grafik (kiri bawah), filter + daftar log (kanan). Filter tersimpan di URL.
import { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Empty, Slider, Tag, Typography } from 'antd'
import type { EChartsOption } from 'echarts'
import { Pause, Play, RotateCcw, TrendingDown, TrendingUp } from 'lucide-react'
import { Polyline } from 'react-leaflet'

import type { AlertRow } from '@/types/alert.types'
import type { HistoryMode } from '@/types/monitoring.types'
import type { Point } from '@/types/telemetry.types'
import { useAlerts } from '@/hooks/useAlerts'
import { useCodeOf } from '@/hooks/useCodeOf'
import { useFuelEvents } from '@/hooks/useFuelEvents'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { usePoints } from '@/hooks/useTelemetry'
import { useZones } from '@/hooks/useZones'
import { alertTypeLabel, brand, equipmentStatusColor } from '@/utils/colors'
import { formatNumber, formatPercent, formatTime } from '@/utils/format'
import { toLocal } from '@/utils/time'
import Chart from '@/components/chart/Chart'
import SlotFilters from '@/components/filters/SlotFilters'
import BaseMap, { FlyTo } from '@/components/map/BaseMap'
import { unitIcon } from '@/components/map/icons'
import { Dot, FitTo, UnitMarker, ZoneLayer } from '@/components/map/layers'
import MapShell from '@/components/map/MapShell'
import InfoRows from '@/components/monitoring/InfoRows'
import ListHeader from '@/components/monitoring/ListHeader'
import styles from '@/components/monitoring/monitoring.module.css'
import PageHeader from '@/components/ui/PageHeader'
import SectionCard from '@/components/ui/SectionCard'

const STEP_MS = 850

export default function HistoryView({ mode }: { mode: HistoryMode }) {
  const { data: zones } = useZones()
  const codeOf = useCodeOf()
  const { values, set, slot } = useSlotFilters({ equipment: undefined })
  const ready = !!slot.equipmentId
  const points = usePoints(slot, { enabled: ready })
  const alerts = useAlerts({ ...slot })
  const fuelEvents = useFuelEvents(slot, ready && mode === 'fuel')
  const rows = useMemo(() => (points.data?.data ?? []).filter((p) => p.latitude != null), [points.data])
  const [idx, setIdx] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [fly, setFly] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  // filter berubah → playback mulai dari awal (disesuaikan saat render, tanpa effect)
  const slotKey = `${slot.date}|${slot.shift}|${slot.equipmentId}`
  const [shownKey, setShownKey] = useState(slotKey)
  if (shownKey !== slotKey) {
    setShownKey(slotKey)
    setIdx(0)
    setPlaying(false)
  }

  useEffect(() => {
    if (!playing) return
    const t = setInterval(() => setIdx((i) => {
      if (i >= rows.length - 1) {
        setPlaying(false)
        return i
      }
      return i + 1
    }), STEP_MS)
    return () => clearInterval(t)
  }, [playing, rows.length])

  useEffect(() => {
    listRef.current?.querySelector(`[data-i="${idx}"]`)?.scrollIntoView({ block: 'nearest' })
  }, [idx])

  // alert yang terjadi pada titik tertentu (penanda di grafik & daftar)
  const alertRows = useMemo(() => {
    const m = new Map<number, AlertRow[]>()
    const list = alerts.data?.data ?? []
    rows.forEach((p, i) => {
      const t = new Date(p.event_time).getTime()
      const hit = list.filter((a) => new Date(a.started_at).getTime() === t)
      if (hit.length) m.set(i, hit)
    })
    return m
  }, [rows, alerts.data])
  const alertAt = useMemo(() => new Map([...alertRows].map(([i, l]) => [i, l.map((a) => alertTypeLabel[a.alert_type] ?? a.alert_type)])), [alertRows])

  const cur: Point | undefined = rows[idx]
  const code = codeOf(slot.equipmentId)
  const path = rows.map((p) => [p.latitude!, p.longitude!] as [number, number])

  const chart = useMemo<EChartsOption>(() => {
    const x = rows.map((p) => toLocal(p.event_time).format('HH:mm'))
    // tanda alert menempel di garis yang sesuai: alert fuel → garis Fuel; overspeed/underspeed/off track → garis Speed.
    // (Mode Fuel hanya punya garis Fuel, jadi semua alert di situ.) rowIndex = urutan titik, dipakai saat tanda diklik.
    const marksOf = (line: 'fuel' | 'speed') => [...alertRows.entries()].flatMap(([i, list]) => {
      const names = list.filter((a) => (mode === 'fuel' || a.alert_type.startsWith('fuel_') ? 'fuel' : 'speed') === line).map((a) => alertTypeLabel[a.alert_type] ?? a.alert_type)
      return names.length ? [{ coord: [i, (line === 'fuel' ? rows[i].fuel_percent : rows[i].speed_kph) ?? 0], value: '!', name: names.join(', '), rowIndex: i }] : []
    })
    const fuel = { name: 'Fuel %', type: 'line' as const, yAxisIndex: mode === 'fuel' ? 0 : 1, showSymbol: false, data: rows.map((p) => p.fuel_percent), itemStyle: { color: brand.accent }, markPoint: { symbol: 'triangle', symbolSize: 12, itemStyle: { color: brand.danger }, data: marksOf('fuel') } }
    const speed = { name: 'Speed (km/h)', type: 'line' as const, showSymbol: false, areaStyle: { opacity: 0.15 }, data: rows.map((p) => p.speed_kph), itemStyle: { color: brand.primary }, markPoint: { symbol: 'triangle', symbolSize: 12, itemStyle: { color: brand.danger }, data: marksOf('speed') } }
    return {
      xAxis: { type: 'category', data: x, axisLabel: { interval: (i: number) => i === 0 || x[i].slice(0, 2) !== x[i - 1]?.slice(0, 2) } },
      yAxis: mode === 'fuel' ? [{ type: 'value', name: 'Fuel %', max: 100 }] : [{ type: 'value', name: 'km/h' }, { type: 'value', name: '%', max: 100 }],
      series: mode === 'fuel' ? [fuel] : [speed, fuel],
      grid: { left: 40, right: mode === 'fuel' ? 20 : 40, top: 36, bottom: 24, containLabel: true },
    }
  }, [rows, alertRows, mode])

  const title = mode === 'fuel' ? 'Fuel History' : 'Position History'
  const panel = (
    <>
      <SlotFilters values={values} onChange={set} />
      <ListHeader title={title} count={rows.length} />
      {ready && points.data?.truncated && <Typography.Text type="warning">Data dipotong — persempit dengan memilih shift.</Typography.Text>}
      <div className={`${styles.list} ${styles.listBox}`} ref={listRef}>
        {!ready && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Pilih unit untuk menampilkan riwayat" />}
        {rows.map((p, i) => (
          <div key={p.id} data-i={i} className={`${styles.listItem} ${i === idx ? styles.listItemActive : ''}`} onClick={() => { setIdx(i); setPlaying(false); setFly((n) => n + 1) }}>
            <div className={styles.listItemTop}>
              <span>{p.segment_name ?? 'Di luar jalur'}</span>
              <span>{formatTime(p.event_time)}</span>
            </div>
            <span className={styles.muted}>
              {mode === 'position' && <>{formatNumber(p.speed_kph)} km/h · </>}Fuel {formatPercent(p.fuel_percent)} ({formatNumber(p.fuel_volume_liter, 1)} L)
            </span>
            {alertAt.get(i)?.map((n) => <Tag key={n} color="red" style={{ marginInlineStart: 6 }}>{n}</Tag>)}
          </div>
        ))}
        {mode === 'fuel' && (fuelEvents.data?.data.length ?? 0) > 0 && (
          <>
            <b style={{ fontSize: 12, marginTop: 6 }}>Kejadian Fuel</b>
            {fuelEvents.data!.data.map((f) => {
              const up = f.event_type === 'increase'
              const color = up ? brand.success : brand.danger
              return (
                <div key={f.id} className={styles.alertCard}>
                  <span className={styles.alertBadge} style={{ background: `${color}1f`, color }}>
                    {up ? <TrendingUp size={18} /> : <TrendingDown size={18} />}
                  </span>
                  <div className={styles.alertBody}>
                    <div className={styles.alertTitle}>
                      <b>{up ? 'Fuel Naik' : 'Fuel Turun'}</b>
                      <span className={styles.alertCode}>{f.fuel_delta_liter > 0 ? '+' : ''}{formatNumber(f.fuel_delta_liter, 2)} L</span>
                    </div>
                    <div className={styles.alertMeta}>
                      <span>{formatTime(f.event_time, false)}</span>
                      {f.description && <span>{f.description}</span>}
                    </div>
                  </div>
                </div>
              )
            })}
          </>
        )}
      </div>
    </>
  )

  return (
    <>
      <PageHeader title={title} subtitle={ready ? `${code} · ${values.date ?? ''}` : undefined} />
      <MapShell
        panel={panel}
        below={
          ready && (
            <SectionCard title={<span style={{ textTransform: 'none' }}>{mode === 'fuel' ? 'Graph View Fuel History' : 'Graph View Position History'}</span>}>
              <Chart height={230} option={chart} loading={points.isFetching} onClick={(e) => { setIdx(e.componentType === 'markPoint' ? ((e.data as { rowIndex?: number })?.rowIndex ?? e.dataIndex) : e.dataIndex); setPlaying(false); setFly((n) => n + 1) }} />
            </SectionCard>
          )
        }
        map={(open) => (
          <>
            <BaseMap resizeKey={open}>
              <ZoneLayer zones={zones} />
              <FitTo points={path} area={zones} fitKey={`${slot.equipmentId}|${slot.date}|${slot.shift}|${path.length > 0}`} />
              {path.length > 1 && <Polyline positions={path} pathOptions={{ color: '#94a3b8', weight: 4, opacity: 0.8 }} />}
              {idx > 0 && <Polyline positions={path.slice(0, idx + 1)} pathOptions={{ color: brand.primary, weight: 5 }} />}
              {rows.map((p, i) =>
                alertAt.has(i) ? (
                  // titik yang ada pelanggarannya (alert) memakai gambar marker "alert", seperti sistem lama
                  <UnitMarker
                    key={p.id}
                    position={[p.latitude!, p.longitude!]}
                    icon={unitIcon({ code, status: p.equipment_status, vessel: p.vessel_status, heading: p.heading_deg, alert: true, breakdown: !!p.is_breakdown })}
                    popup={<InfoRows title={`${code} · ${alertAt.get(i)!.join(', ')}`} d={{ time: p.event_time, speed: p.speed_kph, fuelPercent: p.fuel_percent, segment: p.segment_name, lat: p.latitude, lon: p.longitude }} />}
                    onClick={() => { setIdx(i); setPlaying(false) }}
                  />
                ) : (
                  <Dot
                    key={p.id}
                    position={[p.latitude!, p.longitude!]}
                    radius={3}
                    color={equipmentStatusColor[p.equipment_status ?? 'offline'] ?? '#94a3b8'}
                    onClick={() => { setIdx(i); setPlaying(false) }}
                  />
                ),
              )}
              {cur && (
                <UnitMarker
                  position={[cur.latitude!, cur.longitude!]}
                  icon={unitIcon({ code, status: cur.equipment_status, vessel: cur.vessel_status, heading: cur.heading_deg, alert: alertAt.has(idx), breakdown: !!cur.is_breakdown, selected: true, showLabel: true })}
                  popup={<InfoRows title={code} d={{ time: cur.event_time, speed: cur.speed_kph, fuelLiter: cur.fuel_volume_liter, fuelPercent: cur.fuel_percent, segment: cur.segment_name, status: cur.equipment_status, vessel: cur.vessel_status, engine: cur.is_engine_on, operator: cur.operator_name, lat: cur.latitude, lon: cur.longitude }} />}
                />
              )}
              {/* saat diputar peta mengikuti setiap titik; selain itu hanya saat titik dipilih */}
              <FlyTo position={cur ? [cur.latitude!, cur.longitude!] : null} trigger={playing ? `p${idx}` : `f${fly}`} />
            </BaseMap>
            {rows.length > 0 && (
              <div className={styles.playback}>
                <Button size="small" type="primary" shape="circle" icon={playing ? <Pause size={14} /> : <Play size={14} />} onClick={() => setPlaying(!playing)} aria-label="Putar" />
                <Button size="small" shape="circle" icon={<RotateCcw size={14} />} onClick={() => { setIdx(0); setPlaying(false) }} aria-label="Ulang" />
                <b style={{ minWidth: 44 }}>{cur ? formatTime(cur.event_time, false) : '--:--'}</b>
                <Slider style={{ flex: 1, margin: 0 }} min={0} max={Math.max(rows.length - 1, 0)} value={idx} onChange={(v) => { setIdx(v); setPlaying(false) }} tooltip={{ formatter: (v) => (v != null && rows[v] ? formatTime(rows[v].event_time) : '') }} />
              </div>
            )}
          </>
        )}
      />
    </>
  )
}
