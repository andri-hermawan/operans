// Ringkasan kinerja tiap unit pada shift ini. PA diberi batang warna (hijau ≥ 90, kuning ≥ 75, merah di bawahnya).
import { Badge, Progress, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'

import type { DashboardUnitRow } from '@/types/dashboard.types'
import { brand, equipmentStatusColor, equipmentStatusLabel } from '@/utils/colors'
import { formatNumber } from '@/utils/format'

const paColor = (v: number) => (v >= 90 ? brand.success : v >= 75 ? brand.warning : brand.danger)
const num = (digits: number) => (v: number) => (v ? formatNumber(v, digits) : '-')

const columns: ColumnsType<DashboardUnitRow> = [
  { title: 'Asset ID', dataIndex: 'code', fixed: 'left', width: 100, sorter: (a, b) => a.code.localeCompare(b.code), render: (c: string) => <b>{c}</b> },
  {
    title: 'Status',
    dataIndex: 'status',
    width: 110,
    filters: Object.entries(equipmentStatusLabel).map(([value, text]) => ({ value, text })),
    onFilter: (v, r) => r.status === v,
    render: (s: string) => <Badge color={equipmentStatusColor[s]} text={equipmentStatusLabel[s] ?? s} />,
  },
  { title: 'PA', dataIndex: 'pa', width: 150, sorter: (a, b) => a.pa - b.pa, render: (v: number) => <Progress percent={v} size="small" strokeColor={paColor(v)} format={(p) => `${p}%`} /> },
  { title: 'Running (jam)', dataIndex: 'running', align: 'right', sorter: (a, b) => a.running - b.running, render: num(2) },
  { title: 'Idle (jam)', dataIndex: 'idle', align: 'right', sorter: (a, b) => a.idle - b.idle, render: num(2) },
  { title: 'Breakdown (jam)', dataIndex: 'breakdown', align: 'right', sorter: (a, b) => a.breakdown - b.breakdown, render: num(2) },
  { title: 'Ritase', dataIndex: 'ritase', align: 'right', sorter: (a, b) => a.ritase - b.ritase, render: num(0) },
  { title: 'Produksi (ton)', dataIndex: 'production', align: 'right', sorter: (a, b) => a.production - b.production, render: num(1) },
  { title: 'Avg / Max (km/h)', key: 'speed', align: 'right', render: (_: unknown, r) => (r.avg || r.max ? `${formatNumber(r.avg, 1)} / ${formatNumber(r.max, 0)}` : '-') },
  { title: 'Jarak (km)', dataIndex: 'mileage', align: 'right', sorter: (a, b) => a.mileage - b.mileage, render: num(1) },
  { title: 'BBM turun (L)', dataIndex: 'fuelUsed', align: 'right', sorter: (a, b) => a.fuelUsed - b.fuelUsed, render: num(0) },
  { title: 'Burn (km/L)', dataIndex: 'burn', align: 'right', render: num(2) },
  { title: 'Alert', dataIndex: 'alerts', align: 'right', fixed: 'right', width: 80, sorter: (a, b) => a.alerts - b.alerts, render: (v: number) => (v ? <b style={{ color: brand.danger }}>{v}</b> : '-') },
]

export default function UnitPerformanceTable({ rows, loading }: { rows: DashboardUnitRow[]; loading?: boolean }) {
  return (
    <Table<DashboardUnitRow>
      size="small"
      rowKey="code"
      loading={loading}
      columns={columns}
      dataSource={rows}
      scroll={{ x: 1200 }}
      pagination={{ pageSize: 10, showSizeChanger: false, hideOnSinglePage: true }}
    />
  )
}
