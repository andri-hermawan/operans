// Jenis alert / Alert Category (core-api).
import { core } from '@/services/http/client'
import type { AlertType } from '@/types/alertRule.types'
import type { ListResponse } from '@/types/api.types'

export const listAlertTypes = async () => (await core.get<ListResponse<AlertType>>('/api/v1/alert-types')).data
export const updateAlertType = (code: string, body: Record<string, unknown>) => core.patch(`/api/v1/alert-types/${code}`, body)
