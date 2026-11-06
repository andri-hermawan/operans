// Dokumen SOP & pencarian (layanan AI Brain).

/** Admin project & superadmin selalu boleh membaca; role di bawah diatur per dokumen. */
export type ReaderRole = 'dispatcher' | 'viewer'
export type ProcessingStatus = 'uploaded' | 'processing' | 'ready' | 'failed'

export interface Department {
  id: string
  department_code: string
  department_name: string
}

export interface NewDepartment {
  department_code: string
  department_name: string
}

export interface AiDocument {
  id: string
  department_id: string
  department_code: string
  department_name: string
  title: string
  document_number: string | null
  revision: string | null
  file_name: string
  file_size_bytes: number
  page_count: number | null
  processing_status: ProcessingStatus
  processing_error: string | null
  created_at: string
  /** Role pembaca; hanya diisi untuk admin. */
  roles: ReaderRole[] | null
}

export interface DocumentFilters {
  department_id?: string
}

export interface PageContent {
  document_id: string
  page_number: number
  page_count: number
  content: string
}

export interface SearchHit {
  document_id: string
  title: string
  document_number: string | null
  department_code: string
  department_name: string
  page_number: number
  /** Teks biasa (bukan HTML); kata yang cocok diapit [[ ]]. */
  snippet: string
  score: number
}

export interface SearchResult {
  query: string
  count: number
  hits: SearchHit[]
}

/** Isian form unggah dokumen. */
export interface UploadDocumentInput {
  file: File
  department_id: string
  title: string
  document_number?: string
  revision?: string
  roles: ReaderRole[]
}
