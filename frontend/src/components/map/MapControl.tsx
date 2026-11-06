// Membungkus isi React sebagai kontrol Leaflet di sudut peta. Beberapa kontrol di sudut yang sama tersusun otomatis
// ke bawah menurut urutan pemasangannya (yang lebih dulu di atas), tanpa menghitung jarak manual.
import { useEffect, useMemo, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import L, { type ControlPosition } from 'leaflet'
import { useMap } from 'react-leaflet'

export default function MapControl({ position = 'topleft', children }: { position?: ControlPosition; children: ReactNode }) {
  const map = useMap()
  // wadah dibuat sekali; kontrol Leaflet hanya memasangnya ke sudut peta
  const box = useMemo(() => {
    const div = L.DomUtil.create('div', 'leaflet-control')
    // klik/geser/gulir di tombol tidak boleh ikut menggerakkan peta
    L.DomEvent.disableClickPropagation(div)
    L.DomEvent.disableScrollPropagation(div)
    return div
  }, [])

  useEffect(() => {
    const Control = L.Control.extend({ onAdd: () => box })
    const control = new Control({ position })
    control.addTo(map)
    return () => {
      control.remove()
    }
  }, [map, position, box])

  return createPortal(children, box)
}
