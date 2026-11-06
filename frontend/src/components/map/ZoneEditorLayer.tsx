// Lapisan editor peta area: gambar polygon, geser titik, geser area, hapus (Leaflet-Geoman).
// Layer disimpan di `group`; properti tiap area ada di layer.zoneProps (lihat types/zoneEditor.types.ts).
import { useEffect } from 'react'
import L from 'leaflet'
import '@geoman-io/leaflet-geoman-free'
import '@geoman-io/leaflet-geoman-free/dist/leaflet-geoman.css'
import { useMap } from 'react-leaflet'

import type { EditorLayer } from '@/types/zoneEditor.types'
import type { ZoneCollection } from '@/types/zone.types'
import { newZoneProps, propsOfZone, styleOf } from '@/utils/zoneEditor'

interface Props {
  group: L.FeatureGroup
  /** Peta yang berlaku; dimuat ulang ke editor setiap `loadKey` berubah. */
  zones?: ZoneCollection
  loadKey: number
  /** Boleh menggambar/mengubah (false = hanya melihat). */
  editable: boolean
  selected?: number
  onSelect: (id?: number) => void
  /** Dipanggil saat ada perubahan (gambar, geser, hapus, muat) — untuk memperbarui daftar dan tanda "belum disimpan". */
  onChange: (userEdit: boolean) => void
}

const label = (l: EditorLayer) => l.zoneProps.Segment || '(belum diberi nama)'

export default function ZoneEditorLayer({ group, zones, loadKey, editable, selected, onSelect, onChange }: Props) {
  const map = useMap()

  // layer area ditambahkan ke peta sekali
  useEffect(() => {
    group.addTo(map)
    return () => {
      group.remove()
    }
  }, [map, group])

  /** Pasang properti, gaya, label, dan pendengar klik/ubah pada satu layer. */
  const attach = (l: EditorLayer) => {
    l.setStyle(styleOf(l.zoneProps, false))
    l.bindTooltip(() => label(l), { sticky: true })
    l.on('click', () => onSelect(L.stamp(l)))
    l.on('pm:edit', () => onChange(true))
    l.on('pm:dragend', () => onChange(true))
    l.on('pm:cut', () => onChange(true))
  }

  // muat peta yang berlaku
  useEffect(() => {
    group.clearLayers()
    L.geoJSON((zones ?? { type: 'FeatureCollection', features: [] }) as GeoJSON.FeatureCollection, {
      onEachFeature: (f, layer) => {
        const l = layer as EditorLayer
        l.zoneProps = propsOfZone(f as unknown as ZoneCollection['features'][number])
        attach(l)
        group.addLayer(l)
      },
    })
    onSelect(undefined)
    onChange(false)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- hanya saat peta berlaku / tombol muat ulang berubah
  }, [zones, loadKey, group])

  // toolbar gambar
  useEffect(() => {
    if (!editable) return
    map.pm.addControls({
      position: 'topleft',
      drawPolygon: true,
      drawRectangle: true,
      drawMarker: false,
      drawCircleMarker: false,
      drawPolyline: false,
      drawCircle: false,
      drawText: false,
      editMode: true,
      dragMode: true,
      cutPolygon: false,
      removalMode: true,
      rotateMode: false,
    })
    map.pm.setPathOptions({ color: '#f59e0b', fillColor: '#BEE8FF', fillOpacity: 0.35 })
    const created = ({ layer }: { layer: L.Layer }) => {
      const l = layer as EditorLayer
      l.zoneProps = newZoneProps(group.getLayers() as EditorLayer[])
      attach(l)
      group.addLayer(l)
      onSelect(L.stamp(l))
      onChange(true)
    }
    const removed = ({ layer }: { layer: L.Layer }) => {
      group.removeLayer(layer)
      onSelect(undefined)
      onChange(true)
    }
    map.on('pm:create', created)
    map.on('pm:remove', removed)
    return () => {
      map.off('pm:create', created)
      map.off('pm:remove', removed)
      map.pm.disableDraw()
      map.pm.disableGlobalEditMode()
      map.pm.removeControls()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- pendengar hanya dipasang ulang saat hak ubah berganti
  }, [map, group, editable])

  // tonjolkan area yang dipilih
  useEffect(() => {
    for (const l of group.getLayers() as EditorLayer[]) l.setStyle(styleOf(l.zoneProps, L.stamp(l) === selected))
  }, [group, selected])

  return null
}
