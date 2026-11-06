// Tombol legenda marker di setiap peta (kontrol Leaflet, di bawah tombol reset). Arahkan kursor / ketuk → gambar legenda.
// Posisi bisa diatur lewat prop `position` (BaseMap: `legendPosition`).
import type { ControlPosition } from 'leaflet'
import { Button, Popover } from 'antd'
import { Info } from 'lucide-react'

import legendImage from '@/assets/markers/Legend.png'

import MapControl from './MapControl'

export default function MapLegend({ position = 'topleft' }: { position?: ControlPosition }) {
  return (
    <MapControl position={position}>
      <Popover
        trigger={['hover', 'click']}
        placement={position.endsWith('right') ? 'leftTop' : 'rightTop'}
        title="Legend"
        content={<img src={legendImage} alt="Legend marker" style={{ width: 'min(640px, 80vw)', height: 'auto', display: 'block' }} />}
      >
        <Button icon={<Info size={16} />} aria-label="Legend" style={{ boxShadow: '0 1px 4px rgb(0 0 0 / 0.3)' }} />
      </Popover>
    </MapControl>
  )
}
