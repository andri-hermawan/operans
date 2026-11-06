// Bantuan editor peta area: nilai awal, warna per kategori, dan perakitan GeoJSON untuk diunggah.
import type L from 'leaflet'

import type { EditorLayer, ZoneProps } from '@/types/zoneEditor.types'
import type { ZoneFeature } from '@/types/zone.types'

export const DEFAULT_LINE = '#000000'
export const DEFAULT_FILL = '#BEE8FF'

/** Isian master yang diedit di form; atribut lain di file asli dibawa apa adanya lewat `extra`. */
const KNOWN = ['category', 'segment', 'name', 'hex_line', 'hex_fill', 'priority', 'buff_dist', 'orig_fid', 'code', 'code_2']

/** Area baru: nilai awal seperti master (BUFF_DIST 0, CODE "0", CODE_2 0); ORIG_FID = nomor berikutnya yang belum dipakai. */
export const newZoneProps = (layers: EditorLayer[] = []): ZoneProps => ({
  Category: '',
  Segment: '',
  HEX_Line: DEFAULT_LINE,
  HEX_Fill: DEFAULT_FILL,
  Priority: 0,
  BUFF_DIST: 0,
  ORIG_FID: layers.reduce((m, l) => Math.max(m, l.zoneProps.ORIG_FID + 1), 0),
  CODE: '0',
  CODE_2: 0,
  extra: {},
})

const num = (v: unknown, fallback: number) => (v === null || v === undefined || v === '' || Number.isNaN(Number(v)) ? fallback : Number(v))

/** Properti dari area yang sedang berlaku (core-api) ke isian master. */
export const propsOfZone = (f: ZoneFeature): ZoneProps => ({
  Category: f.properties.category,
  Segment: f.properties.name,
  HEX_Line: f.properties.line_color ?? DEFAULT_LINE,
  HEX_Fill: f.properties.fill_color ?? DEFAULT_FILL,
  Priority: f.properties.priority ?? 0,
  BUFF_DIST: num(f.properties.attributes?.BUFF_DIST, 0),
  ORIG_FID: f.properties.source_fid ?? num(f.properties.attributes?.ORIG_FID, 0),
  CODE: String(f.properties.attributes?.CODE ?? '0'),
  CODE_2: num(f.properties.attributes?.CODE_2, 0),
  extra: Object.fromEntries(Object.entries(f.properties.attributes ?? {}).filter(([k]) => !KNOWN.includes(k.toLowerCase()))),
})

export const styleOf = (p: ZoneProps, selected: boolean): L.PathOptions => ({
  color: selected ? '#f59e0b' : p.HEX_Line,
  weight: selected ? 4 : 2,
  fillColor: p.HEX_Fill,
  fillOpacity: selected ? 0.5 : 0.35,
})

/** Warna kategori yang sudah dipakai area lain (huruf besar/kecil tidak dibedakan); undefined bila kategori baru. */
export function colorOfCategory(layers: EditorLayer[], category: string, except?: EditorLayer): Pick<ZoneProps, 'HEX_Line' | 'HEX_Fill'> | undefined {
  const key = category.trim().toLowerCase()
  const hit = key ? layers.find((l) => l !== except && l.zoneProps.Category.trim().toLowerCase() === key) : undefined
  return hit && { HEX_Line: hit.zoneProps.HEX_Line, HEX_Fill: hit.zoneProps.HEX_Fill }
}

/** Area yang belum lengkap (Category / Segment kosong). */
export const incomplete = (layers: EditorLayer[]) => layers.filter((l) => !l.zoneProps.Category.trim() || !l.zoneProps.Segment.trim())

/** Semua area → GeoJSON FeatureCollection dengan properti persis seperti file master (dipakai jalur unggah yang sama). */
export function toFeatureCollection(layers: EditorLayer[]) {
  return {
    type: 'FeatureCollection',
    features: layers.map((l) => {
      const p = l.zoneProps
      // urutan & nama sama dengan file master
      const properties: Record<string, unknown> = {
        Category: p.Category.trim(),
        Segment: p.Segment.trim(),
        BUFF_DIST: p.BUFF_DIST,
        ORIG_FID: p.ORIG_FID,
        CODE: p.CODE,
        CODE_2: p.CODE_2,
        HEX_Line: p.HEX_Line,
        HEX_Fill: p.HEX_Fill,
        Priority: p.Priority,
        ...p.extra,
      }
      return { type: 'Feature', geometry: l.toGeoJSON().geometry, properties }
    }),
  }
}
