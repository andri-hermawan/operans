// User (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { UserSummary } from '@/types/user.types'

/** projectId kosong = semua user (superadmin). */
export const listUsers = async (projectId?: string) =>
  (await core.get<ListResponse<UserSummary>>('/api/v1/users', { params: { project_id: projectId } })).data
export const createUser = (body: Record<string, unknown>) => core.post('/api/v1/users', body)
export const updateUser = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/users/${id}`, body)
export const resetUserPassword = (id: string, newPassword: string) => core.post(`/api/v1/users/${id}/reset-password`, { new_password: newPassword })
