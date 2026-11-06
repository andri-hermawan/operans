// Lapisan peta bersama: area (zona), penanda unit, titik, legenda.
import { useEffect, useMemo, useRef, type ReactNode } from 'react'
import L, { type LatLngBoundsExpression } from 'leaflet'
import { CircleMarker, GeoJSON, Marker, Popup, Tooltip, useMap } from 'react-leaflet'

import type { MapView } from '@/types/telemetry.types'
import type { ZoneCollection, ZoneFeature } from '@/types/zone.types'
import { brand } from '@/utils/colors'

import styles from './map.module.css'
import { useMapHome } from './mapHome'

/** Lama animasi terbang peta (detik 0,8) — popup dibuka sesudahnya. */
const FLY_MS = 800

/** Area peta project. `highlight` = nama area yang ditonjolkan (geofence). */
export function ZoneLayer({ zones, highlight, onSelect, labels = false }: { zones?: ZoneCollection; highlight?: string; onSelect?: (name: string) => void; labels?: boolean }) {
  if (!zones) return null
  return (
    <GeoJSON
      key={`${zones.features.length}-${highlight}-${labels}`}
      data={zones as GeoJSON.FeatureCollection}
      style={(f) => {
        const p = (f as ZoneFeature).properties
        const on = highlight && p.name === highlight
        return { color: on ? brand.accent : (p.line_color ?? '#334155'), weight: on ? 3 : 1, fillColor: p.fill_color ?? '#94a3b8', fillOpacity: on ? 0.35 : 0.12 }
      }}
      onEachFeature={(f, layer) => {
        const p = (f as ZoneFeature).properties
        layer.bindTooltip(`${p.name}${p.category ? ` · ${p.category}` : ''}`, { sticky: !labels, permanent: labels, direction: 'center', className: styles.zoneLabel })
        if (onSelect) layer.on('click', () => onSelect(p.name))
      }}
    />
  )
}

export function UnitMarker({ position, icon, popup, onClick, open }: { position: [number, number]; icon: L.DivIcon; popup?: ReactNode; onClick?: () => void; open?: boolean }) {
  const ref = useRef<L.Marker>(null)
  // open berubah → popup dibuka (setelah peta selesai terbang ke unit) atau ditutup
  useEffect(() => {
    const m = ref.current
    if (!m || open === undefined) return
    if (!open) {
      m.closePopup()
      return
    }
    const t = setTimeout(() => m.openPopup(), FLY_MS + 50)
    return () => clearTimeout(t)
  }, [open])
  return (
    <Marker ref={ref} position={position} icon={icon} eventHandlers={onClick ? { click: onClick } : undefined}>
      {popup && <Popup minWidth={230}>{popup}</Popup>}
    </Marker>
  )
}

/** Titik kecil berwarna (riwayat, sebaran alert, heatmap). Klik/sentuh membuka popup. */
export function Dot({ position, color, radius = 6, popup, tooltip, onClick, selected }: { position: [number, number]; color: string; radius?: number; popup?: ReactNode; tooltip?: string; onClick?: () => void; selected?: boolean }) {
  return (
    <CircleMarker center={position} radius={selected ? radius + 4 : radius} pathOptions={{ color: selected ? '#111827' : '#ffffff', weight: selected ? 3 : 1, fillColor: color, fillOpacity: 0.9 }} eventHandlers={onClick ? { click: onClick } : undefined}>
      {tooltip && <Tooltip>{tooltip}</Tooltip>}
      {popup && <Popup minWidth={220}>{popup}</Popup>}
    </CircleMarker>
  )
}

/**
 * Atur tampilan peta agar semua titik (atau seluruh area) terlihat. Hanya SEKALI per `fitKey`
 * (mis. per tanggal/shift/unit) — pembaruan live tidak membuat peta melompat.
 */
export function FitTo({ points, area, fitKey = '', padding = 40 }: { points?: [number, number][]; area?: ZoneCollection; fitKey?: string; padding?: number }) {
  const map = useMap()
  const done = useRef<string>(undefined)
  const bounds = useMemo<LatLngBoundsExpression | undefined>(() => {
    if (points && points.length) return L.latLngBounds(points)
    if (area?.features.length) return L.geoJSON(area as GeoJSON.FeatureCollection).getBounds()
    return undefined
  }, [points, area])
  const homeRef = useMapHome()
  // tombol Reset View selalu tahu batas terbaru
  useEffect(() => {
    if (homeRef && bounds) homeRef.current = { bounds, padding }
  }, [homeRef, bounds, padding])
  useEffect(() => {
    if (!bounds || done.current === fitKey) return
    done.current = fitKey
    map.fitBounds(bounds, { padding: [padding, padding], maxZoom: 17 })
  }, [map, bounds, padding, fitKey])
  return null
}

/** Legenda kecil di pojok peta. */
export function Legend({ items, title }: { items: { color: string; label: string; ring?: boolean }[]; title?: string }) {
  return (
    <div className={styles.legend}>
      {title && <div className={styles.legendTitle}>{title}</div>}
      {items.map((i) => (
        <div key={i.label} className={styles.legendRow}>
          <span className={styles.legendDot} style={i.ring ? { border: `3px solid ${i.color}`, background: '#fff' } : { background: i.color }} />
          {i.label}
        </div>
      ))}
    </div>
  )
}

/** Kembali ke tampilan awal (batas FitTo) setiap kali `trigger` berubah, dan tutup popup. Sama dengan tombol Reset View. */
export function GoHome({ trigger }: { trigger: number }) {
  const map = useMap()
  const homeRef = useMapHome()
  useEffect(() => {
    if (!trigger) return
    map.closePopup()
    const h = homeRef?.current
    if (h) map.flyToBounds(h.bounds, { padding: [h.padding, h.padding], maxZoom: 17, duration: 0.8 })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- hanya saat trigger berubah
  }, [trigger])
  return null
}

/** Melaporkan bagian peta yang terlihat (batas + zoom) setiap selesai digeser/di-zoom, dengan jeda singkat. */
export function MapViewWatcher({ onChange }: { onChange: (view: MapView) => void }) {
  const map = useMap()
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined
    const report = () => {
      clearTimeout(timer)
      timer = setTimeout(() => {
        const b = map.getBounds()
        const r = (v: number) => Math.round(v * 10000) / 10000
        onChange({ bbox: [r(b.getWest()), r(b.getSouth()), r(b.getEast()), r(b.getNorth())], zoom: map.getZoom() })
      }, 350)
    }
    map.on('moveend zoomend', report)
    report()
    return () => {
      clearTimeout(timer)
      map.off('moveend zoomend', report)
    }
  }, [map, onChange])
  return null
}
