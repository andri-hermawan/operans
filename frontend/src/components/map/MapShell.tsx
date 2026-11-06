import { useState, type ReactNode } from 'react'
import { Button, Grid, Tooltip } from 'antd'
import { PanelRightClose, PanelRightOpen } from 'lucide-react'

import styles from '@/components/components.module.css'

interface Props {
  /** Isi peta: biasanya <BaseMap resizeKey={open}>…</BaseMap> — terima `open` lewat fungsi. */
  map: (panelOpen: boolean) => ReactNode
  /** Panel kanan (filter, daftar, ringkasan). */
  panel: ReactNode
  /** Isi di bawah peta pada kolom kiri (mis. grafik riwayat). */
  below?: ReactNode
  /** Lebar panel di desktop: '1fr' (peta 3fr) atau px. Sama dengan FMS lama. */
  panelWidth?: string
  /** Tinggi peta di HP. */
  mobileMapHeight?: number
}

/**
 * Tata letak "peta + panel samping" yang dipakai tracking, geofence, peta sebaran, riwayat posisi/fuel,
 * speed heatmap. Desktop: dua kolom, panel bisa disembunyikan. HP: peta di atas, lalu isi bawah & panel.
 */
export default function MapShell({ map, panel, below, panelWidth = '1fr', mobileMapHeight = 380 }: Props) {
  const screens = Grid.useBreakpoint()
  const mobile = screens.lg === false
  const [open, setOpen] = useState(true)

  if (mobile) {
    return (
      <div className={styles.mapShell} style={{ gridTemplateColumns: '1fr' }}>
        <div className={styles.mapPane} style={{ height: mobileMapHeight, minHeight: mobileMapHeight }}>{map(true)}</div>
        {below}
        <div className={styles.mapPanel}>{panel}</div>
      </div>
    )
  }

  const mapCol = panelWidth.endsWith('fr') ? '3fr' : 'minmax(0, 1fr)'
  return (
    <div className={styles.mapShell} style={{ gridTemplateColumns: open ? `${mapCol} ${panelWidth}` : '1fr', height: 'calc(100vh - 60px - 48px - 56px)' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12, minHeight: 0, minWidth: 0 }}>
        <div className={styles.mapPane} style={{ flex: 1 }}>
          <Tooltip title={open ? 'Sembunyikan panel' : 'Tampilkan panel'} placement="left">
            <Button className={styles.mapToggle} icon={open ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />} onClick={() => setOpen(!open)} />
          </Tooltip>
          {map(open)}
        </div>
        {below}
      </div>
      {open && <div className={styles.mapPanel}>{panel}</div>}
    </div>
  )
}
