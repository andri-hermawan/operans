// Satu kartu di Speed Heatmap List: lencana kecepatan rata-rata kotak, jumlah lintasan, dan rata-rata kosong / bermuatan.
import { Route, Users } from 'lucide-react'

import type { HeatCell } from '@/types/telemetry.types'
import { speedColor } from '@/utils/colors'
import { formatNumber } from '@/utils/format'
import styles from '@/components/monitoring/monitoring.module.css'

export default function HeatCellCard({ cell: c, active, onClick }: { cell: HeatCell; active: boolean; onClick: () => void }) {
  const color = speedColor(c.avg_speed)
  const kph = (v: number | null) => (v == null ? '-' : formatNumber(v, 1))
  return (
    <div className={`${styles.alertCard} ${active ? styles.listItemActive : ''}`} onClick={onClick}>
      <span className={`${styles.alertBadge} ${styles.speedBadge}`} style={{ background: `${color}1f`, color, borderColor: color }}>
        <b>{formatNumber(c.avg_speed, 0)}</b>
        <small>km/h</small>
      </span>
      <div className={styles.alertBody}>
        <div className={styles.alertTitle}>
          <span className={styles.alertCode}>Maks {formatNumber(c.max_speed, 0)} km/h</span>
        </div>
        <div className={styles.alertMeta}>
          <span><Route size={12} /> {formatNumber(c.points)} titik</span>
          <span><Users size={12} /> {c.units} unit</span>
        </div>
        <div className={styles.alertMeta}>
          <span>Kosong {kph(c.avg_speed_empty)}</span>
          <span>Bermuatan {kph(c.avg_speed_loaded)}</span>
        </div>
      </div>
    </div>
  )
}
