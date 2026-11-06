// Jadwal laporan otomatis (core-api): daftar, ubah, kirim sekarang, lihat PDF, riwayat.
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Schedule, ScheduleLog } from '@/types/reportSchedule.types'

const PATH = '/api/v1/report-schedules'

export const listReportSchedules = async (projectId: string) => (await core.get<ListResponse<Schedule>>(PATH, { params: { project_id: projectId } })).data
export const createReportSchedule = (body: Record<string, unknown>) => core.post(PATH, body)
export const updateReportSchedule = (id: string, body: Record<string, unknown>) => core.patch(`${PATH}/${id}`, body)
export const deleteReportSchedule = (id: string) => core.del(`${PATH}/${id}`)
export const listScheduleLogs = async (id: string) => (await core.get<ListResponse<ScheduleLog>>(`${PATH}/${id}/logs`)).data
export const sendScheduleNow = (id: string) => core.post(`${PATH}/${id}/send-now`)
export const previewSchedulePdf = (id: string) => core.get<Blob>(`${PATH}/${id}/preview`, { blob: true })
