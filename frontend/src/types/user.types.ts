// User & keanggotaan project (core-api).

export interface UserSummary {
  id: string
  email: string
  name: string
  phone_number: string | null
  is_superadmin: boolean
  is_active: boolean
  last_login_at: string | null
  created_at: string
  memberships: { project_id: string; project_code: string; project_name: string; role: string }[]
}

/** Baris tabel User: user + role di project aktif. */
export type UserRow = UserSummary & { role?: string }
