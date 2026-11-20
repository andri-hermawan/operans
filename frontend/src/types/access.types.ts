// Hak akses halaman & laporan yang diatur superadmin (core-api /api/v1/access, ADR 0018).
import type { ProjectRole } from './auth.types'

export interface AccessRule {
  subject_type: 'role' | 'user'
  role?: ProjectRole
  user_id?: string
  page_key: string
  allowed: boolean
}

/** allowed null = hapus aturan (kembali ke bawaan / ikut role). */
export interface AccessChange {
  page_key: string
  allowed: boolean | null
}

export interface AccessSaveInput {
  project_id: string
  subject_type: 'role' | 'user'
  role?: ProjectRole
  user_id?: string
  changes: AccessChange[]
}
