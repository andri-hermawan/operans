// Rencana produksi core-api (ADR 0014): daftar per tahun (+ bulan), tambah, ubah, hapus. Impor Excel lewat /imports/:type.
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { MonthlyPlan, YearlyPlan } from '@/types/plan.types'

const planApi = <T extends { id: number }>(path: string) => ({
  /** Satu tahun (dan satu bulan bila month diisi). */
  list: async (projectId: string, year?: string, month?: string) =>
    (await core.get<ListResponse<T>>(path, { params: { project_id: projectId, year, month } })).data,
  create: (body: Record<string, unknown>) => core.post(path, body),
  update: (id: number, body: Record<string, unknown>) => core.patch(`${path}/${id}`, body),
  remove: (id: number) => core.del(`${path}/${id}`),
})

export const yearlyPlanApi = planApi<YearlyPlan>('/api/v1/yearly-plans')
export const monthlyPlanApi = planApi<MonthlyPlan>('/api/v1/monthly-plans')
