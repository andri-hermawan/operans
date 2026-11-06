// Nilai referensi core-api (kategori breakdown, status perbaikan, ...).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { ReferenceValue } from '@/types/referenceValue.types'

export const listReferenceValues = async (group: string) =>
  (await core.get<ListResponse<ReferenceValue>>('/api/v1/reference-values', { params: { group } })).data
