// Unit / alat (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Equipment } from '@/types/equipment.types'

export const listEquipment = async (projectId: string, includeInactive = false) =>
  (await core.get<ListResponse<Equipment>>('/api/v1/equipment', { params: { project_id: projectId, include_inactive: includeInactive || undefined } })).data
export const createEquipment = (body: Record<string, unknown>) => core.post('/api/v1/equipment', body)
export const updateEquipment = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/equipment/${id}`, body)
