// Riwayat & unggah peta area GeoJSON (core-api).
import { core } from '@/services/http/client'
import type { ListResponse } from '@/types/api.types'
import type { GeoImportResult, LayerImport } from '@/types/geoImport.types'

export const listGeoImports = async (projectId: string) =>
  (await core.get<ListResponse<LayerImport>>('/api/v1/geo/imports', { params: { project_id: projectId } })).data

/** dryRun = pratinjau (diperiksa server, belum berlaku). */
export const uploadGeoImport = (projectId: string | undefined, body: FormData, dryRun: boolean) =>
  core.post<{ data: GeoImportResult }>('/api/v1/geo/imports', body, { params: { project_id: projectId, dry_run: dryRun || undefined } })

export const downloadGeoImportFile = (id: string) => core.get<Blob>(`/api/v1/geo/imports/${id}/file`, { blob: true })
