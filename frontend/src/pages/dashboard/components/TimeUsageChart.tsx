// Pemakaian jam armada selama shift: berapa jam bekerja (running), diam (idle), rusak (breakdown), dan sisanya.
import type { EChartsOption } from 'echarts'

import type { TimeUsage } from '@/types/dashboard.types'
import { equipmentStatusColor } from '@/utils/colors'
import Chart from '@/components/chart/Chart'

export default function TimeUsageChart({ usage }: { usage: TimeUsage }) {
  const total = usage.running + usage.idle + usage.breakdown + usage.other
  const option: EChartsOption = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} jam ({d}%)' },
    legend: { bottom: 0, type: 'plain' },
    title: { text: `${total}`, subtext: 'jam tersedia', left: 'center', top: '34%', textStyle: { fontSize: 24, fontWeight: 700 }, subtextStyle: { fontSize: 11 } },
    series: [
      {
        type: 'pie',
        radius: ['52%', '74%'],
        center: ['50%', '45%'],
        label: { show: false },
        data: [
          { name: 'Running', value: usage.running, itemStyle: { color: equipmentStatusColor.running } },
          { name: 'Idle', value: usage.idle, itemStyle: { color: equipmentStatusColor.idle } },
          { name: 'Breakdown', value: usage.breakdown, itemStyle: { color: equipmentStatusColor.breakdown } },
          { name: 'Stop / lainnya', value: usage.other, itemStyle: { color: '#cbd5e1' } },
        ],
      },
    ],
  }
  return <Chart option={option} height={250} />
}
