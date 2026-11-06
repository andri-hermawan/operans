// Tipe login & sesi (endpoint /auth/* core-api, ADR 0002).

export type ProjectRole = 'admin' | 'dispatcher' | 'viewer'
export type Role = 'superadmin' | ProjectRole

export interface Membership {
  project_id: string
  project_code: string
  project_name: string
  role: ProjectRole
}

export interface SessionUser {
  id: string
  email: string
  name: string
  is_superadmin: boolean
}

export interface Session {
  access_token: string
  expires_in: number
  access_expires_at: string
  refresh_token: string
  refresh_expires_at: string
  user: SessionUser
  project: Membership | null
}

export interface Profile {
  user: SessionUser
  active_project_id: string | null
  memberships: Membership[]
}

export type AuthStatus = 'restoring' | 'signedOut' | 'signedIn'

/** Aksi & sumber hak akses (tabel hak ada di utils/permissions.ts). */
export type Action = 'view' | 'create' | 'update' | 'delete' | 'import' | 'export'

/** Isian form login. */
export interface LoginForm {
  email: string
  password: string
}

/** Isian form ganti password (profil). */
export interface PasswordForm {
  current_password: string
  new_password: string
  confirm: string
}

/** Kelompok sumber data untuk hak akses (tabelnya di utils/permissions.ts). */
export type Resource = 'monitoring' | 'report' | 'daily' | 'master' | 'organization' | 'documents'
