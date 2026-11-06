// Produksi per unit: batang = ton, garis = ritase. Hanya unit yang berproduksi, urut dari yang terbanyak.
import { Empty } from 'antd'
import type { EChartsOption } from 'echarts'

import type { DashboardUnitRow } from '@/types/dashboard.types'
import { brand } from '@/utils/colors'
import Chart from '@/components/chart/Chart'

const TOP = 15

export default function UnitProductionChart({ rows }: { rows: DashboardUnitRow[] }) {
  const top = rows.filter((r) => r.production > 0 || r.ritase > 0).sort((a, b) => b.production - a.production).slice(0, TOP)
  if (!top.length) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Belum ada tiket timbang pada periode ini" />
  const option: EChartsOption = {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { bottom: 0 },
    grid: { left: 40, right: 40, top: 24, bottom: 48, containLabel: true },
    xAxis: { type: 'category', data: top.map((r) => r.code), axisLabel: { fontSize: 11, rotate: top.length > 8 ? 40 : 0 } },
    yAxis: [
      { type: 'value', name: 'Ton', splitLine: { lineStyle: { type: 'dashed', color: '#e5e5e5' } } },
      { type: 'value', name: 'Rit', minInterval: 1, splitLine: { show: false } },
    ],
    series: [
      { name: 'Produksi (ton)', type: 'bar', data: top.map((r) => r.production), itemStyle: { color: brand.primary, borderRadius: [4, 4, 0, 0] }, barMaxWidth: 28 },
      { name: 'Ritase', type: 'line', yAxisIndex: 1, data: top.map((r) => r.ritase), itemStyle: { color: brand.accent }, lineStyle: { width: 3 }, symbolSize: 7 },
    ],
  }
  return <Chart option={option} height={280} />
}
