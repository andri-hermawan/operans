// Perusahaan (core-api).

export interface Company {
  id: string
  company_code: string
  company_name: string
  is_active: boolean
  project_count: number
  created_at: string
}
