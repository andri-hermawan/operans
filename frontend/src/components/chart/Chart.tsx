import { lazy, Suspense, useMemo } from 'react'
import { Skeleton } from 'antd'
import type { EChartsOption } from 'echarts'

import { brand } from '@/utils/colors'

const ReactECharts = lazy(() => import('echarts-for-react'))

/** Gaya dasar semua grafik (huruf, warna, grid, tooltip) — halaman cukup memberi series & sumbu. */
const BASE: EChartsOption = {
  color: [brand.primary, brand.accent, brand.success, brand.danger, brand.info, '#7c3aed', '#0891b2'],
  textStyle: { fontFamily: "Inter, 'Segoe UI', system-ui, sans-serif", color: brand.text },
  grid: { left: 48, right: 48, top: 36, bottom: 32, containLabel: true },
  tooltip: { trigger: 'axis', confine: true },
  legend: { top: 0, type: 'scroll' },
}

interface Props {
  option: EChartsOption
  height?: number | string
  loading?: boolean
  onClick?: (params: { dataIndex: number; seriesName?: string; componentType?: string; data?: unknown }) => void
}

/** Grafik ECharts standar, ikut menyesuaikan lebar wadah. */
export default function Chart({ option, height = 300, loading, onClick }: Props) {
  const merged = useMemo(() => ({ ...BASE, ...option }), [option])
  return (
    <Suspense fallback={<Skeleton.Node active style={{ width: '100%', height }} />}>
      <ReactECharts
        option={merged}
        style={{ height, width: '100%' }}
        notMerge
        showLoading={loading}
        onEvents={onClick ? { click: onClick } : undefined}
      />
    </Suspense>
  )
}
