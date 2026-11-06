// Impor Excel ke core-api: unduh template, pratinjau (dry_run), simpan (ADR 0007).
import { core } from '@/services/http/client'
import type { ImportResult, ImportType } from '@/types/import.types'

export const uploadImport = (type: ImportType, body: FormData, projectId: string | undefined, dryRun: boolean) =>
  core.post<{ data: ImportResult }>(`/api/v1/imports/${type}`, body, { params: { project_id: projectId, dry_run: dryRun || undefined } })

export const downloadImportTemplate = (type: ImportType) => core.get<Blob>(`/api/v1/imports/${type}/template`, { blob: true })
