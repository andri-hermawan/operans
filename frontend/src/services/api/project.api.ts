// Project / site (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Project } from '@/types/project.types'

export const listProjects = async () => (await core.get<ListResponse<Project>>('/api/v1/projects')).data
export const createProject = (body: Record<string, unknown>) => core.post('/api/v1/projects', body)
export const updateProject = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/projects/${id}`, body)
/** Ubah role user di project (user dicari lewat email). */
export const setProjectMemberRole = (projectId: string, email: string, role: string) =>
  core.put(`/api/v1/projects/${projectId}/members`, { email, role })
/** Keluarkan user dari project (akunnya tetap ada). */
export const removeProjectMember = (projectId: string, userId: string) => core.del(`/api/v1/projects/${projectId}/members/${userId}`)
