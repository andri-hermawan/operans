// Isi laporan Alert Summary — dipakai halaman laporan DAN halaman cetak PDF (jadwal laporan).
// Susunan sama dengan sistem lama: kartu ringkasan per kategori (Empty / Loaded), lokasi kejadian per segmen,
// lalu frekuensi kejadian dan jumlah unit per jam untuk tiap kategori.
import { useMemo } from 'react'
import { Alert, Col, Empty, Row } from 'antd'
import type { EChartsOption } from 'echarts'
import { Fuel, Gauge, MapPinOff, TrendingDown, type LucideIcon } from 'lucide-react'

import type { AlertRow } from '@/types/alert.types'
import type { Shift } from '@/types/shift.types'
import { useZones } from '@/hooks/useZones'
import { alertTypeColor } from '@/utils/colors'
import { shiftHours, toLocal } from '@/utils/time'
import Chart from '@/components/chart/Chart'
import SectionCard from '@/components/ui/SectionCard'

interface Props {
  alerts: AlertRow[]
  shift?: Shift
  shifts: Shift[]
  /** false di halaman cetak (grafik langsung jadi). */
  animate?: boolean
  /** true di halaman cetak PDF: kolom berukuran tetap (dua kolom seperti layar lebar). Saat dicetak, breakpoint layar dibaca dari
   *  lebar kertas (±1060 px), bukan lebar susunan, sehingga tanpa ini PDF jatuh ke susunan satu kolom. */
  fixedLayout?: boolean
}

/** Empat kategori laporan (seperti sistem lama): dua jenis Fuel Decrease digabung. Warna dari utils/colors. */
const CATEGORIES: { key: string; label: string; types: string[]; color: string; icon: LucideIcon }[] = [
  { key: 'fuel', label: 'Fuel Decrease', types: ['fuel_decrease_engine_on', 'fuel_decrease_engine_off'], color: alertTypeColor.fuel_decrease_engine_on, icon: Fuel },
  { key: 'off_track', label: 'Off Track', types: ['off_track'], color: alertTypeColor.off_track, icon: MapPinOff },
  { key: 'overspeed', label: 'Overspeed', types: ['overspeed'], color: alertTypeColor.overspeed, icon: Gauge },
  { key: 'underspeed', label: 'Underspeed', types: ['underspeed'], color: alertTypeColor.underspeed, icon: TrendingDown },
]

const OUTSIDE = 'Di luar jalur'
const pad = (h: number) => String(h).padStart(2, '0')
const grid = { splitLine: { lineStyle: { type: 'dashed' as const, color: '#e5e5e5' } } }

export default function AlertSummaryReport({ alerts, shift, shifts, animate = true, fixedLayout = false }: Props) {
  // lebar kolom (dari 24): di layar satu kolom pada layar sempit & `span` pada layar lebar; di PDF selalu `span`
  const cols = (span: number) => (fixedLayout ? { span } : { xs: 24, xl: span })
  const hours = useMemo(() => {
    if (shift) return shiftHours(shift)
    const first = shifts[0]
    return first ? shiftHours({ start_time: first.start_time, end_time: first.start_time }) : Array.from({ length: 24 }, (_, i) => i)
  }, [shift, shifts])
  const { data: zones } = useZones()
  const ranges = hours.map((h) => `${pad(h)}-${pad((h + 1) % 24)}`)

  const cards = CATEGORIES.map((c) => {
    const list = alerts.filter((a) => c.types.includes(a.alert_type))
    const side = (v: string) => {
      const l = list.filter((a) => a.vessel_status === v)
      return { events: l.length, units: new Set(l.map((a) => a.equipment_id)).size }
    }
    return { ...c, total: list.length, units: new Set(list.map((a) => a.equipment_id)).size, empty: side('empty'), loaded: side('loaded') }
  })

  // semua zona di peta (meski tanpa alert) + "Di luar jalur" bila ada alert di luar zona; diurutkan agar yang terbanyak di atas
  const segments = useMemo(() => {
    const count = new Map<string, number>()
    for (const f of zones?.features ?? []) count.set(f.properties.name, 0)
    for (const a of alerts) {
      const k = a.segment_name ?? OUTSIDE
      count.set(k, (count.get(k) ?? 0) + 1)
    }
    return [...count.entries()].sort((x, y) => x[1] - y[1] || y[0].localeCompare(x[0])).map(([name]) => name)
  }, [alerts, zones])

  const hourOf = (a: AlertRow) => toLocal(a.started_at).hour()
  const inCat = (c: (typeof CATEGORIES)[number]) => alerts.filter((a) => c.types.includes(a.alert_type))

  const locationOption: EChartsOption = {
    animation: animate,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { show: false },
    grid: { left: 8, right: 24, top: 8, bottom: 8, containLabel: true },
    xAxis: { type: 'value', minInterval: 1, ...grid },
    yAxis: { type: 'category', data: segments, axisLabel: { fontSize: 11, interval: 0 }, axisTick: { show: false } },
    series: CATEGORIES.map((c) => ({
      name: c.label,
      type: 'bar' as const,
      stack: 'all',
      barMaxWidth: 22,
      itemStyle: { color: c.color },
      label: { show: true, position: 'inside' as const, color: '#fff', fontSize: 11, formatter: (p: { value?: unknown }) => (p.value ? String(p.value) : '') },
      data: segments.map((s) => inCat(c).filter((a) => (a.segment_name ?? OUTSIDE) === s).length),
    })),
  }

  /** Grafik kecil per kategori (satu grafik per baris selebar kartu): satu batang per jam shift. `count` menghitung nilai tiap jam. */
  const mini = (c: (typeof CATEGORIES)[number], count: (list: AlertRow[]) => number): EChartsOption => ({
    animation: animate,
    tooltip: { trigger: 'axis' },
    legend: { show: false },
    grid: { left: 8, right: 8, top: 22, bottom: 4, containLabel: true },
    xAxis: { type: 'category', data: ranges, axisLabel: { fontSize: 9, interval: 0 }, axisTick: { show: false } },
    yAxis: { type: 'value', minInterval: 1, ...grid },
    series: [
      {
        name: c.label,
        type: 'bar',
        barMaxWidth: 20,
        itemStyle: { color: c.color, borderRadius: [4, 4, 0, 0] },
        label: { show: true, position: 'top', fontSize: 10, formatter: (p: { value?: unknown }) => (p.value ? String(p.value) : '') },
        data: hours.map((h) => count(inCat(c).filter((a) => hourOf(a) === h))),
      },
    ],
  })

  const miniGrid = (title: string, count: (list: AlertRow[]) => number) => (
    <SectionCard title={title}>
      <Row gutter={[16, 16]}>
        {CATEGORIES.map((c) => (
          <Col key={c.key} span={24}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, fontWeight: 600, marginBottom: 2 }}>
              <span style={{ width: 10, height: 10, borderRadius: '50%', background: c.color }} />
              {c.label}
            </div>
            <Chart option={mini(c, count)} height={170} />
          </Col>
        ))}
      </Row>
    </SectionCard>
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* tanpa alert: laporan tetap lengkap (nilai 0) supaya PDF tidak tampil kosong */}
      {!alerts.length && <Alert type="info" showIcon message="Tidak ada alert pada tanggal/shift ini" />}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 12 }}>
        {cards.map((c) => (
          <div key={c.key} style={{ background: '#fff', border: '1px solid #e5e7eb', borderRadius: 12, padding: 14 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ display: 'grid', placeItems: 'center', width: 40, height: 40, borderRadius: 10, background: `${c.color}26`, color: c.color }}>
                <c.icon size={20} />
              </span>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 13, color: '#64748b' }}>{c.label}</div>
                <div style={{ fontSize: 24, fontWeight: 700, lineHeight: 1.1 }}>
                  {c.total}
                  <span style={{ fontSize: 12, fontWeight: 500, color: '#64748b', marginLeft: 4 }}>event · {c.units} DT</span>
                </div>
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginTop: 12 }}>
              {[
                { label: 'Empty', v: c.empty },
                { label: 'Loaded', v: c.loaded },
              ].map(({ label, v }) => (
                <div key={label} style={{ background: `${c.color}14`, borderRadius: 8, padding: '8px 10px' }}>
                  <div style={{ fontSize: 11, color: '#64748b', fontWeight: 600 }}>{label}</div>
                  <div style={{ fontSize: 14, fontWeight: 700, color: '#1f2937' }}>
                    {v.events} <span style={{ fontWeight: 500, fontSize: 12 }}>event</span> · {v.units} <span style={{ fontWeight: 500, fontSize: 12 }}>DT</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <Row gutter={[12, 12]} align="top">
        <Col {...cols(10)}>
          <SectionCard title="Abnormal Event Location">
            {alerts.length ? <Chart option={locationOption} height={Math.max(360, segments.length * 28 + 30)} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Tidak ada kejadian" />}
          </SectionCard>
        </Col>
        <Col {...cols(14)}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {miniGrid('Hourly Abnormal Activity · Event Frequency', (l) => l.length)}
            {miniGrid('Hourly Abnormal Activity · Equipment Quantity', (l) => new Set(l.map((a) => a.equipment_id)).size)}
          </div>
        </Col>
      </Row>
    </div>
  )
}
