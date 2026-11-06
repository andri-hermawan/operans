// Shift kerja project (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { Shift, ShiftRow } from '@/types/shift.types'

/** Shift aktif project, dipakai sebagai pilihan filter & penentu tanggal operasional. */
export const listShifts = async (projectId: string) =>
  (await core.get<ListResponse<Shift>>('/api/v1/shifts', { params: { project_id: projectId } })).data

/** Semua shift project untuk halaman pengaturan (termasuk yang nonaktif). */
export const listShiftSettings = async (projectId: string) =>
  (await core.get<ListResponse<ShiftRow>>('/api/v1/shifts/manage', { params: { project_id: projectId } })).data
export const createShift = (body: Record<string, unknown>) => core.post('/api/v1/shifts', body)
export const updateShift = (id: string, body: Record<string, unknown>) => core.patch(`/api/v1/shifts/${id}`, body)
