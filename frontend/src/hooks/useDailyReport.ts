import { useQuery } from '@tanstack/react-query'

import { listDailyReport } from '@/services/api/report.api'
import type { DailyReportKind, ReportPeriod } from '@/types/report.types'

import { useProjectKey } from './useProjectKey'

const useDaily = (kind: DailyReportKind, p: ReportPeriod, enabled = true) => {
  const { pid, key } = useProjectKey('report', kind, p)
  return useQuery({
    queryKey: key,
    enabled: !!pid && !!p.from && !!p.to && enabled,
    queryFn: () => listDailyReport(kind, pid!, p),
  })
}

export const useWeighbridgeDaily = (p: ReportPeriod, enabled?: boolean) => useDaily('weighbridge-daily', p, enabled)
export const useBreakdownDaily = (p: ReportPeriod, enabled?: boolean) => useDaily('breakdown-daily', p, enabled)
