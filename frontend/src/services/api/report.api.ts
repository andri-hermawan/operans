// Rekap laporan harian dari core-api (weighbridge & breakdown per unit per tanggal operasional).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { DailyCell, DailyReportKind, ExecutiveEmplacementRow, ExecutiveHaulingRow, ReportPeriod } from '@/types/report.types'

/** Produksi (Total TLS Out) & inventory (Sisa Cargo) per partner pada `date`; produksi juga MTD dan YTD. */
export const getExecutiveEmplacement = async (projectId: string, date: string) =>
  (await core.get<ListResponse<ExecutiveEmplacementRow>>('/api/v1/reports/executive-emplacement', { params: { project_id: projectId, date } })).data

/** Produksi per customer pada `date` (harian), bulan berjalan (MTD), dan tahun berjalan (YTD). */
export const getExecutiveHauling = async (projectId: string, date: string) =>
  (await core.get<ListResponse<ExecutiveHaulingRow>>('/api/v1/reports/executive-hauling', { params: { project_id: projectId, date } })).data

export const listDailyReport = async (kind: DailyReportKind, projectId: string, p: ReportPeriod) =>
  (
    await core.get<ListResponse<DailyCell>>(`/api/v1/reports/${kind}`, {
      params: { project_id: projectId, date_from: p.from, date_to: p.to, shift_id: p.shiftId, equipment_code: p.codes?.join(',') },
    })
  ).data
