// Dua grafik alert: jumlah per jenis (batang mendatar, jenis dibaca dari label) dan sebaran per jam shift (kapan alert paling sering).
import type { EChartsOption } from 'echarts'

import type { AlertHourRow } from '@/types/dashboard.types'
import { alertTypeColor, alertTypeLabel } from '@/utils/colors'
import Chart from '@/components/chart/Chart'

export function AlertByTypeChart({ data }: { data: { type: string; count: number }[] }) {
  const option: EChartsOption = {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 8, right: 36, top: 8, bottom: 8, containLabel: true },
    xAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { type: 'dashed', color: '#e5e5e5' } } },
    yAxis: { type: 'category', inverse: true, data: data.map((d) => alertTypeLabel[d.type] ?? d.type), axisTick: { show: false } },
    series: [
      {
        type: 'bar',
        barWidth: 18,
        data: data.map((d) => ({ value: d.count, itemStyle: { color: alertTypeColor[d.type], borderRadius: [0, 6, 6, 0] } })),
        label: { show: true, position: 'right', formatter: (p) => (p.value === 0 ? '' : String(p.value)) },
      },
    ],
  }
  return <Chart option={option} height={250} />
}

export function AlertByHourChart({ data }: { data: AlertHourRow[] }) {
  const option: EChartsOption = {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { bottom: 0, type: 'scroll' },
    grid: { left: 36, right: 12, top: 16, bottom: 48, containLabel: true },
    xAxis: { type: 'category', data: data.map((d) => d.hour), axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', minInterval: 1, splitLine: { lineStyle: { type: 'dashed', color: '#e5e5e5' } } },
    series: Object.keys(alertTypeLabel).map((t) => ({
      name: alertTypeLabel[t],
      type: 'bar' as const,
      stack: 'alert',
      itemStyle: { color: alertTypeColor[t] },
      data: data.map((d) => d.counts[t] ?? 0),
    })),
  }
  return <Chart option={option} height={250} />
}
