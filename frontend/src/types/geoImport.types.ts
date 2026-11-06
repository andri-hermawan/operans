// Unggah peta area (GeoJSON) project dan hasil pratinjaunya (core-api, ADR 0003).

export interface LayerImport {
  id: string
  file_name: string
  feature_count: number
  note: string | null
  imported_by_name: string | null
  imported_at: string
  is_current: boolean
}
export interface GeoIssue {
  feature?: number
  name?: string
  field?: string
  message: string
}
export interface GeoImportResult {
  feature_count: number
  valid_features: number
  invalid_features: number
  fixed_geometries: number
  replaced_zones: number
  categories: { category: string; count: number }[]
  errors: GeoIssue[]
  warnings: GeoIssue[]
}
