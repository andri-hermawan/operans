// Tombol "kembali ke tampilan awal" (di atas tombol Legend): terbang lagi ke batas yang dipakai FitTo
// (unit aktif / area). Berguna setelah pengguna menggeser atau memperbesar peta ke satu unit.
import { Button, Tooltip } from 'antd'
import { Crosshair } from 'lucide-react'
import { useMap } from 'react-leaflet'

import { DEFAULT_CENTER, DEFAULT_ZOOM } from '@/utils/map'

import MapControl from './MapControl'
import { useMapHome } from './mapHome'

export default function ResetViewButton() {
  const map = useMap()
  const homeRef = useMapHome()
  const reset = () => {
    const h = homeRef?.current
    if (h) map.flyToBounds(h.bounds, { padding: [h.padding, h.padding], maxZoom: 17, duration: 0.6 })
    else map.flyTo(DEFAULT_CENTER, DEFAULT_ZOOM, { duration: 0.6 })
  }
  return (
    <MapControl position="topleft">
      <Tooltip title="Kembali ke tampilan awal" placement="right">
        <Button icon={<Crosshair size={16} />} aria-label="Reset view" onClick={reset} style={{ boxShadow: '0 1px 4px rgb(0 0 0 / 0.3)' }} />
      </Tooltip>
    </MapControl>
  )
}
