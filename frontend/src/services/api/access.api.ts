// Hak akses halaman & laporan (core-api). Membaca hak sendiri: semua user. Melihat semua aturan & menyimpan: superadmin.
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { AccessRule, AccessSaveInput } from '@/types/access.types'

export const accessApi = {
  /** Perubahan dari aturan bawaan untuk user ini di project aktif (kunci halaman → boleh/tidak). */
  mine: async () => (await core.get<{ data: Record<string, boolean> }>('/api/v1/access/mine')).data,
}

export const listAccessRules = async (projectId: string) =>
  (await core.get<ListResponse<AccessRule>>('/api/v1/access/rules', { params: { project_id: projectId } })).data
export const saveAccessRules = (body: AccessSaveInput) => core.put('/api/v1/access/rules', body)
