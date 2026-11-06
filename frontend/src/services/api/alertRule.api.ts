// Aturan ambang alert per project/zona (core-api).
import { core } from '@/services/http/client'
import type { AlertRule } from '@/types/alertRule.types'
import type { ListResponse } from '@/types/api.types'

export const listAlertRules = async (projectId: string) =>
  (await core.get<ListResponse<AlertRule>>('/api/v1/alert-rules', { params: { project_id: projectId } })).data
export const createAlertRule = (body: Record<string, unknown>) => core.post('/api/v1/alert-rules', body)
export const updateAlertRule = (id: string, body: Record<string, unknown>) => core.put(`/api/v1/alert-rules/${id}`, body)
export const deleteAlertRule = (id: string) => core.del(`/api/v1/alert-rules/${id}`)
