// Dashboard — satu shift diceritakan lewat kartu dan grafik, berurutan: kondisi armada → produksi → waktu kerja →
// breakdown → pergerakan & BBM → alert → rincian per unit. Semua mengikuti filter Tanggal + Shift.
import { Col, Row, Tag } from 'antd'
import { Activity, Clock, Fuel, Gauge, MapPin, Package, Play, Radio, Route, Scale, ShieldCheck, Timer, Truck, TriangleAlert, Users, Wrench } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { useDashboardData } from '@/hooks/useDashboardData'
import { useLiveSocket } from '@/hooks/useLiveSocket'
import { useSlotFilters } from '@/hooks/useSlotFilters'
import { alertTypeColor, alertTypeLabel, brand, equipmentStatusColor } from '@/utils/colors'
import { formatNumber } from '@/utils/format'
import { dayjs } from '@/utils/time'
import SlotFilters from '@/components/filters/SlotFilters'
import PageHeader from '@/components/ui/PageHeader'

import { AlertByHourChart, AlertByTypeChart } from './components/AlertCharts'
import BreakdownFeed from './components/BreakdownFeed'
import Chapter from './components/Chapter'
import FleetStatusBar from './components/FleetStatusBar'
import RecentAlerts from './components/RecentAlerts'
import TimeUsageChart from './components/TimeUsageChart'
import UnitPerformanceTable from './components/UnitPerformanceTable'
import UnitProductionChart from './components/UnitProductionChart'
import styles from './dashboard.module.css'

const LIVE_TEXT = { connecting: 'Menyambung…', live: 'Live', offline: 'Terputus' } as const
const n = (v: number, d = 0) => formatNumber(v, d)

export default function DashboardPage() {
  const liveStatus = useLiveSocket()
  const navigate = useNavigate()
  const { values, set, slot } = useSlotFilters()
  const d = useDashboardData(slot)
  const { kpi } = d

  const shiftText = d.shift ? `${d.shift.shift_name} (${d.shift.start_time}–${d.shift.end_time})` : 'Semua shift'
  const paColor = kpi.pa >= 90 ? brand.success : kpi.pa >= 75 ? brand.warning : brand.danger
  const paTone = kpi.pa >= 90 ? 'good' : kpi.pa >= 75 ? 'warn' : 'bad'
  const bdUnits = d.rows.filter((r) => r.breakdown > 0).length
  const topAlert = [...d.alertByType].sort((a, b) => b.count - a.count)[0]

  return (
    <div className={styles.page}>
      <PageHeader
        title="Dashboard"
        subtitle={`${dayjs(slot.date).format('dddd, D MMMM YYYY')} · ${shiftText}`}
        extra={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
            <Tag color={liveStatus === 'live' ? 'success' : liveStatus === 'connecting' ? 'processing' : 'error'}>{LIVE_TEXT[liveStatus]}</Tag>
            <div style={{ width: 360, maxWidth: '100%' }}>
              <SlotFilters asset={false} values={values} onChange={set} />
            </div>
          </div>
        }
      />

      <Chapter
        no={1}
        title="Kondisi armada saat ini"
        stats={[
          { icon: Truck, label: 'Unit terpantau', value: n(kpi.units) },
          { icon: Radio, label: 'Online', value: n(kpi.online), color: brand.info },
          { icon: Play, label: 'Running', value: n(kpi.running), color: equipmentStatusColor.running },
        ]}
      >
        <FleetStatusBar byStatus={d.byStatus} total={kpi.units} />
      </Chapter>

      <Chapter
        no={2}
        title="Produksi"
        stats={[
          { icon: Package, label: 'Ritase', value: n(kpi.ritase), unit: 'rit', color: brand.info },
          { icon: Scale, label: 'Produksi', value: n(kpi.tonnes, 1), unit: 'ton', color: brand.success },
          { icon: Gauge, label: 'Payload rata-rata', value: n(kpi.payload, 1), unit: 'ton/rit', color: brand.accent },
          { icon: Users, label: 'Unit berproduksi', value: n(d.rows.filter((r) => r.ritase > 0).length) },
        ]}
      >
        <UnitProductionChart rows={d.rows} />
      </Chapter>

      <Chapter
        no={3}
        title="Waktu kerja armada"
        tone={paTone}
        stats={[
          { icon: ShieldCheck, label: 'Physical Availability', value: n(kpi.pa), unit: '%', color: paColor },
          { icon: Activity, label: 'Utilization', value: n(kpi.utilization), unit: '%', color: equipmentStatusColor.running },
          { icon: Timer, label: 'Jam running', value: n(d.timeUsage.running, 1), unit: 'jam', color: equipmentStatusColor.running },
          { icon: Clock, label: 'Jam idle', value: n(d.timeUsage.idle, 1), unit: 'jam', color: equipmentStatusColor.idle },
        ]}
      >
        <TimeUsageChart usage={d.timeUsage} />
      </Chapter>

      <Chapter
        no={4}
        title="Breakdown"
        tone={bdUnits ? 'bad' : 'good'}
        stats={[
          { icon: Wrench, label: 'Jam breakdown', value: n(kpi.breakdownHours, 1), unit: 'jam', color: equipmentStatusColor.breakdown },
          { icon: Truck, label: 'Unit terdampak', value: n(bdUnits), color: bdUnits ? brand.danger : undefined },
          { icon: TriangleAlert, label: 'Kejadian', value: n(d.breakdowns.filter((b) => b.status !== 'ready').length), color: brand.warning },
        ]}
      >
        <BreakdownFeed items={d.breakdowns} />
      </Chapter>

      <Chapter
        no={5}
        title="Pergerakan dan BBM"
        stats={[
          { icon: Route, label: 'Jarak tempuh', value: n(kpi.mileage), unit: 'km', color: brand.primary },
          { icon: Gauge, label: 'Kecepatan rata-rata', value: n(kpi.avgSpeed, 1), unit: 'km/h', color: brand.info },
          { icon: Fuel, label: 'BBM turun', value: n(kpi.fuelUsed), unit: 'L', color: brand.warning },
          { icon: Fuel, label: 'BBM terisi', value: n(kpi.fuelFilled), unit: 'L', color: brand.success },
          { icon: MapPin, label: 'Burn ratio', value: n(kpi.burn, 2), unit: 'km/L', color: brand.accent },
        ]}
      />

      <Chapter
        no={6}
        title="Alert"
        tone={kpi.alerts ? 'warn' : 'good'}
        stats={[
          { icon: TriangleAlert, label: 'Total alert', value: n(kpi.alerts), color: kpi.alerts ? brand.danger : brand.muted },
          ...d.alertByType.filter((a) => a.count > 0).map((a) => ({ icon: TriangleAlert, label: alertTypeLabel[a.type], value: n(a.count), color: alertTypeColor[a.type] })),
        ]}
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={9}>
            <p className={styles.subTitle}>Per jenis</p>
            <AlertByTypeChart data={d.alertByType} />
          </Col>
          <Col xs={24} xl={9}>
            <p className={styles.subTitle}>Per jam shift</p>
            <AlertByHourChart data={d.alertByHour} />
          </Col>
          <Col xs={24} xl={6}>
            <p className={styles.subTitle}>
              Terbaru {topAlert?.count ? '· ' : ''}
              <a onClick={() => navigate(`/alert?from=${slot.date}&to=${slot.date}`)}>lihat semua</a>
            </p>
            <RecentAlerts items={d.recentAlerts} />
          </Col>
        </Row>
      </Chapter>

      <Chapter no={7} title="Rincian per unit">
        <UnitPerformanceTable rows={d.rows} loading={d.loading} />
      </Chapter>
    </div>
  )
}
