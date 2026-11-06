// Editor peta area (gambar polygon di sistem). Nama isian sama dengan properti GeoJSON master (ADR 0003).
import type L from 'leaflet'

export interface ZoneProps {
  Category: string
  Segment: string
  HEX_Line: string
  HEX_Fill: string
  Priority: number
  BUFF_DIST: number
  ORIG_FID: number
  CODE: string
  CODE_2: number
  /** Atribut lain dari file asli (tidak ditampilkan, tetap dibawa saat disimpan). */
  extra: Record<string, unknown>
}

/** Layer polygon di editor beserta properti area-nya. */
export type EditorLayer = L.Polygon & { zoneProps: ZoneProps }
