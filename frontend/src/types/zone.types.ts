// Peta area (GeoJSON) dari core-api.

export interface ZoneFeature {
  type: 'Feature'
  id: string
  geometry: GeoJSON.Geometry
  properties: { id: string; category: string; name: string; line_color: string | null; fill_color: string | null; priority: number; source_fid: number | null; area_m2: number; attributes?: Record<string, unknown> }
}
export interface ZoneCollection {
  type: 'FeatureCollection'
  features: ZoneFeature[]
}
