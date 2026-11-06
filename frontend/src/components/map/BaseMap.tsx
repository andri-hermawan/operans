import { useEffect, useRef, type ReactNode } from 'react'
import type { ControlPosition, LatLngExpression } from 'leaflet'
import { LayersControl, MapContainer, TileLayer, useMap } from 'react-leaflet'

import { BASE_LAYERS, DEFAULT_CENTER, DEFAULT_ZOOM } from '@/utils/map'

import { MapHomeContext, type HomeView } from './mapHome'
import MapLegend from './MapLegend'
import ResetViewButton from './ResetViewButton'

interface Props {
  center?: LatLngExpression
  zoom?: number
  children?: ReactNode
  /** Nilai yang bila berubah membuat peta menghitung ulang ukurannya (mis. panel dibuka/ditutup). */
  resizeKey?: unknown
  /** Sudut tombol Legend (bawaan kiri atas, di bawah kontrol zoom & lapisan). */
  legendPosition?: ControlPosition
}

/** Leaflet perlu diberi tahu bila ukuran wadahnya berubah. */
function AutoResize({ resizeKey }: { resizeKey?: unknown }) {
  const map = useMap()
  useEffect(() => {
    const t = setTimeout(() => map.invalidateSize(), 220)
    return () => clearTimeout(t)
  }, [map, resizeKey])
  useEffect(() => {
    const ro = new ResizeObserver(() => map.invalidateSize())
    ro.observe(map.getContainer())
    return () => ro.disconnect()
  }, [map])
  return null
}

/** Peta dasar semua halaman: pilihan Peta Jalan / Satelit, tombol Reset View & Legend, ukuran mengikuti wadah. */
export default function BaseMap({ center = DEFAULT_CENTER, zoom = DEFAULT_ZOOM, children, resizeKey, legendPosition }: Props) {
  const homeRef = useRef<HomeView | undefined>(undefined)
  // preferCanvas: ribuan titik riwayat/heatmap tetap ringan
  return (
    <MapHomeContext.Provider value={homeRef}>
      <MapContainer center={center} zoom={zoom} style={{ height: '100%', width: '100%' }} zoomControl preferCanvas>
        <LayersControl position="topleft">
          <LayersControl.BaseLayer checked name={BASE_LAYERS.satellite.name}>
            <TileLayer url={BASE_LAYERS.satellite.url} attribution={BASE_LAYERS.satellite.attribution} maxZoom={20} maxNativeZoom={18} />
          </LayersControl.BaseLayer>
          <LayersControl.BaseLayer name={BASE_LAYERS.street.name}>
            <TileLayer url={BASE_LAYERS.street.url} attribution={BASE_LAYERS.street.attribution} maxZoom={20} maxNativeZoom={19} />
          </LayersControl.BaseLayer>
        </LayersControl>
        <AutoResize resizeKey={resizeKey} />
        <ResetViewButton />
        <MapLegend position={legendPosition} />
        {children}
      </MapContainer>
    </MapHomeContext.Provider>
  )
}

/** Terbang ke titik setiap kali `trigger` berubah (pengganti FlyToAlert/FlyToLogMarker/FlyToPoint lama). */
export function FlyTo({ position, zoom = 17, trigger }: { position?: [number, number] | null; zoom?: number; trigger?: unknown }) {
  const map = useMap()
  useEffect(() => {
    if (position) map.flyTo(position, zoom, { duration: 0.8 })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- hanya saat trigger berubah
  }, [trigger])
  return null
}
