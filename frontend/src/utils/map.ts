// Bantuan peta.

/** Link Google Maps untuk satu titik (dipakai di tabel, popup, laporan). */
export const googleMapsUrl = (lat?: number | null, lon?: number | null): string | undefined =>
  lat == null || lon == null ? undefined : `https://www.google.com/maps?q=${lat},${lon}`

/** "−3.470277, 103.871217" (6 desimal seperti sistem lama). */
export const formatCoordinate = (lat?: number | null, lon?: number | null): string =>
  lat == null || lon == null ? '-' : `${lat.toFixed(6)}, ${lon.toFixed(6)}`

/** Pusat peta bawaan bila belum ada data (area CHR). */
export const DEFAULT_CENTER: [number, number] = [-3.487, 103.869]
export const DEFAULT_ZOOM = 14

/** Layer dasar peta. */
export const BASE_LAYERS = {
  street: {
    name: 'Peta Jalan',
    url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
    attribution: '&copy; OpenStreetMap',
  },
  satellite: {
    name: 'Satelit',
    url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
    attribution: 'Tiles &copy; Esri',
  },
} as const

/**
 * Buang titik "pencilan" (mis. unit yang posisi terakhirnya jauh di luar tambang atau 0,0) supaya peta tidak
 * ter-zoom-out ke seluruh dunia. Titik dipertahankan bila ≤ `maxKm` dari titik tengah (median) semua titik.
 */
export function withoutOutliers(points: [number, number][], maxKm = 25): [number, number][] {
  if (points.length < 3) return points
  const median = (vs: number[]) => [...vs].sort((a, b) => a - b)[Math.floor(vs.length / 2)]
  const lat0 = median(points.map((p) => p[0]))
  const lon0 = median(points.map((p) => p[1]))
  const kmPerLon = 111.32 * Math.cos((lat0 * Math.PI) / 180)
  const kept = points.filter(([lat, lon]) => Math.hypot((lat - lat0) * 110.57, (lon - lon0) * kmPerLon) <= maxKm)
  return kept.length ? kept : points
}
