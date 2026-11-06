// Dokumen SOP & pencarian (AI Brain). Jawaban layanan ini apa adanya (array/objek), bukan { data: [...] }.
import { ai } from '@/services/http/client'
import type {
  AiDocument,
  Department,
  DocumentFilters,
  NewDepartment,
  PageContent,
  ReaderRole,
  SearchResult,
  UploadDocumentInput,
} from '@/types/aiDocument.types'

export const listDepartments = () => ai.get<Department[]>('/api/v1/departments')
export const createDepartment = (body: NewDepartment) => ai.post<Department>('/api/v1/departments', body)

export const listAiDocuments = (f: DocumentFilters) => ai.get<AiDocument[]>('/api/v1/documents', { params: { ...f } })

export const uploadAiDocument = (input: UploadDocumentInput) => {
  const form = new FormData()
  form.append('file', input.file)
  form.append('department_id', input.department_id)
  form.append('title', input.title)
  if (input.document_number) form.append('document_number', input.document_number)
  if (input.revision) form.append('revision', input.revision)
  input.roles.forEach((r) => form.append('roles', r))
  return ai.post<AiDocument>('/api/v1/documents', form)
}

export const getDocumentPage = (id: string, page: number) => ai.get<PageContent>(`/api/v1/documents/${id}/pages/${page}`)

export const getDocumentPageImage = (id: string, page: number) =>
  ai.get<Blob>(`/api/v1/documents/${id}/pages/${page}/image`, { blob: true })

/** Proses ulang dokumen yang gagal/tertahan (admin) memakai file yang sudah tersimpan. */
export const reprocessDocument = (id: string) => ai.post<AiDocument>(`/api/v1/documents/${id}/reprocess`)

/** PDF asli untuk dicetak — hanya superadmin (server menolak role lain). */
export const getDocumentFile = (id: string) => ai.get<Blob>(`/api/v1/documents/${id}/file`, { blob: true })

export const setDocumentRoles = (id: string, roles: ReaderRole[]) => ai.put<AiDocument>(`/api/v1/documents/${id}/roles`, { roles })

/** "Tarik" dokumen: hilang dari semua daftar & pencarian (riwayat tetap di database). */
export const deactivateDocument = (id: string) => ai.del(`/api/v1/documents/${id}`)

export const searchDocuments = (q: string, f: DocumentFilters, signal?: AbortSignal) =>
  ai.get<SearchResult>('/api/v1/search', { params: { q, ...f }, signal })
