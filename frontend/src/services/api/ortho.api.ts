// Daftar versi citra ortho. Berkas statis publik (bukan API backend), jadi tanpa token.
import type { OrthoIndex, OrthoVersion } from '@/types/ortho.types'

/** Alamat dasar tile; Vite (dev) dan Nginx (produksi) menyajikan folder `ortho/tiles` di sini. */
export const ORTHO_BASE = '/ortho'

export const orthoTileUrl = (v: OrthoVersion) => `${ORTHO_BASE}/${v.id}/{z}/{x}/{y}.${v.ext}`

/** Kosong bila belum ada ortho (berkas tidak ditemukan) — peta tetap jalan tanpa layer ortho. */
export async function listOrthoVersions(): Promise<OrthoVersion[]> {
  try {
    const res = await fetch(`${ORTHO_BASE}/index.json`, { cache: 'no-cache' })
    if (!res.ok || !(res.headers.get('content-type') ?? '').includes('json')) return []
    const data = (await res.json()) as OrthoIndex
    return Array.isArray(data.versions) ? data.versions : []
  } catch {
    return []
  }
}
