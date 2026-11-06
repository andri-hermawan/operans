// Perusahaan (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Company } from '@/types/company.types'

export const listCompanies = async () => (await core.get<ListResponse<Company>>('/api/v1/companies')).data
export const createCompany = (body: Record<string, unknown>) => core.post('/api/v1/companies', body)
export const updateCompany = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/companies/${id}`, body)
