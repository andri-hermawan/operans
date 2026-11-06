// Report — katalog kartu laporan (dua kelompok seperti FMS lama: Raw Data & Report).
import { useState } from 'react'
import { Card, Col, Empty, Input, Row, Typography } from 'antd'
import {
  AlertTriangle,
  BarChart3,
  Boxes,
  CalendarCheck,
  ChartNoAxesCombined,
  ClipboardList,
  Fuel,
  Gauge,
  History,
  PlayCircle,
  Scale,
  Search,
  Truck,
  Wrench,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/stores/auth.store'
import type { ReportMenuItem } from '@/types/report.types'
import { brand } from '@/utils/colors'
import PageHeader from '@/components/ui/PageHeader'

const ic = (C: typeof Truck) => <C size={26} color={brand.primary} />

const GROUPS: { title: string; items: ReportMenuItem[] }[] = [
  {
    title: 'Raw Data',
    items: [
      { path: '/report/report-equipment-logs', title: 'Asset History', desc: 'Data mentah alat per tanggal & shift', icon: ic(History) },
      { path: '/report/report-fuel-history', title: 'Fuel History', desc: 'Kejadian isi & turun BBM per periode', icon: ic(Fuel) },
      { path: '/alert', title: 'Abnormal Alert History', desc: 'Semua alert per periode', icon: ic(AlertTriangle) },
      { path: '/report/playback-unit-history', title: 'Playback Unit History', desc: 'Putar ulang rute unit', icon: ic(PlayCircle) },
      { path: '/graphic', title: 'Equipment Graphic', desc: 'Availability unit & grafik speed/fuel', icon: ic(ChartNoAxesCombined) },
    ],
  },
  {
    title: 'Report',
    items: [
      { path: '/report/report-ritase-daily', title: 'Ritase', desc: 'Jumlah rit per unit per hari', icon: ic(Truck) },
      { path: '/report/report-payload-daily', title: 'Payload', desc: 'Rata-rata muatan (ton/rit)', icon: ic(Scale) },
      { path: '/report/report-production-daily', title: 'Production', desc: 'Total produksi (ton)', icon: ic(Gauge) },
      { path: '/report/report-breakdown-daily', title: 'Breakdown', desc: 'Jam breakdown per unit', icon: ic(Wrench) },
      { path: '/report/report-pa-daily', title: 'Physical Availability', desc: 'PA % per unit per hari', icon: ic(CalendarCheck) },
      { path: '/report/equipment-shift-performance', title: 'Equipment Shift Performance', desc: 'Kinerja tiap unit dalam satu shift', icon: ic(Gauge), roles: ['superadmin', 'admin'] },
    ],
  },
  {
    // laporan yang bisa dijadwalkan kirim PDF lewat email (jadwal diatur di System Data → Report Schedule)
    title: 'Report Schedule',
    items: [
      { path: '/report/report-alert-summary', title: 'Alert Summary', desc: 'Ringkasan alert per jenis, segmen, jam', icon: ic(BarChart3) },
      { path: '/report/report-executive-hauling', title: 'Executive Summary Report Hauling', desc: 'Produksi per customer: Daily, MTD, YTD', icon: ic(ClipboardList) },
      { path: '/report/report-executive-emplacement', title: 'Executive Summary Report Emplacement', desc: 'Produksi (TLS Out) & inventory per partner', icon: ic(Boxes) },
    ],
  },
]

export default function ReportHub() {
  const navigate = useNavigate()
  const role = useAuthStore((s) => s.role)
  const [q, setQ] = useState('')
  const term = q.trim().toLowerCase()

  return (
    <>
      <PageHeader title="Report" extra={<Input allowClear prefix={<Search size={14} />} placeholder="Cari laporan…" value={q} onChange={(e) => setQ(e.target.value)} style={{ width: 240 }} />} />
      {GROUPS.map((g) => {
        const items = g.items.filter((i) => (!i.roles || (role && i.roles.includes(role))) && `${i.title} ${i.desc}`.toLowerCase().includes(term))
        if (!items.length) return null
        return (
          <div key={g.title} style={{ marginBottom: 20 }}>
            <Typography.Title level={5} style={{ color: brand.primaryDark }}>{g.title}</Typography.Title>
            <Row gutter={[12, 12]}>
              {items.map((i) => (
                <Col key={`${i.path}-${i.title}`}xs={24} sm={12} lg={8} xl={6}>
                  <Card hoverable size="small" onClick={() => navigate(i.path)} style={{ height: '100%' }}>
                    <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                      {i.icon}
                      <div>
                        <b>{i.title}</b>
                        <div style={{ fontSize: 12, color: brand.muted }}>{i.desc}</div>
                      </div>
                    </div>
                  </Card>
                </Col>
              ))}
            </Row>
          </div>
        )
      })}
      {GROUPS.every((g) => !g.items.some((i) => `${i.title} ${i.desc}`.toLowerCase().includes(term))) && <Empty description="Laporan tidak ditemukan" />}
    </>
  )
}
