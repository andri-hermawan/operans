// Hourly Traffic Chart (gaya sistem lama): batang IN (hijau muda) & OUT (oranye muda) per jam shift, legenda di bawah.
import { useMemo } from 'react'
import { Empty } from 'antd'
import type { EChartsOption } from 'echarts'

import Chart from '@/components/chart/Chart'
import styles from '@/components/monitoring/monitoring.module.css'
import SectionCard from '@/components/ui/SectionCard'

import type { HourlyRow } from './HourlyPassingSummary'

const barLabel = { show: true, position: 'top' as const, fontSize: 11, color: '#333', formatter: (p: { value?: unknown }) => (p.value === 0 ? '' : String(p.value)) }
const dashed = { show: true, lineStyle: { type: 'dashed' as const, color: '#e5e5e5', width: 1 } }

/** `hours` = SEMUA jam shift (termasuk yang kosong) agar sumbu lengkap. */
export default function HourlyTrafficChart({ hours, emptyText = 'Belum ada data' }: { hours: HourlyRow[]; emptyText?: string }) {
  const option = useMemo<EChartsOption>(
    () => ({
      tooltip: { trigger: 'axis' },
      legend: { bottom: 4, left: 'center', top: undefined, type: 'plain', textStyle: { fontSize: 12 } },
      grid: { left: 45, right: 15, top: 20, bottom: 40, containLabel: true },
      xAxis: { type: 'category', data: hours.map((h) => h.hour), axisTick: { alignWithLabel: true }, axisLabel: { fontSize: 11 }, splitLine: dashed },
      yAxis: { type: 'value', minInterval: 1, axisLabel: { fontSize: 11 }, splitLine: dashed },
      series: [
        { name: 'IN', type: 'bar', data: hours.map((h) => h.in), barWidth: 16, label: barLabel, itemStyle: { color: '#C6E0B3', borderRadius: [4, 4, 0, 0] } },
        { name: 'OUT', type: 'bar', data: hours.map((h) => h.out), barWidth: 16, label: barLabel, itemStyle: { color: '#F9CBAC', borderRadius: [4, 4, 0, 0] } },
      ],
    }),
    [hours],
  )
  return (
    <SectionCard title="Hourly Traffic Chart" className={`${styles.fillCard} ${styles.fillCol}`} styles={{ body: { padding: 8 } }}>
      {hours.length === 0 ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={emptyText} /> : <Chart option={option} height="100%" />}
    </SectionCard>
  )
}
