// Satu kartu di Abnormal Alert List: lencana ikon berwarna di kiri, jenis alert + kode unit, lalu info kecil (jam, kecepatan, jalur).
import { Fuel, Gauge, MapPin, MapPinOff, TrendingDown, type LucideIcon } from 'lucide-react'
import { Clock } from 'lucide-react'

import type { AlertRow } from '@/types/alert.types'
import { alertTypeColor, alertTypeLabel } from '@/utils/colors'
import { formatTime, formatUnit } from '@/utils/format'
import styles from '@/components/monitoring/monitoring.module.css'

const ICONS: Record<string, LucideIcon> = {
  overspeed: Gauge,
  underspeed: TrendingDown,
  off_track: MapPinOff,
  fuel_decrease_engine_on: Fuel,
  fuel_decrease_engine_off: Fuel,
}

interface Props {
  alert: AlertRow & { code: string }
  active: boolean
  onClick: () => void
}

export default function AlertCard({ alert: a, active, onClick }: Props) {
  const color = alertTypeColor[a.alert_type] ?? '#8A8A8A'
  const Icon = ICONS[a.alert_type] ?? Gauge
  return (
    <div className={`${styles.alertCard} ${active ? styles.listItemActive : ''}`} onClick={onClick}>
      <span className={styles.alertBadge} style={{ background: `${color}26`, color }}>
        <Icon size={18} />
      </span>
      <div className={styles.alertBody}>
        <div className={styles.alertTitle}>
          <b>{alertTypeLabel[a.alert_type] ?? a.alert_type}</b>
          <span className={styles.alertCode}>{a.code}</span>
        </div>
        <div className={styles.alertMeta}>
          <span><Clock size={12} /> {formatTime(a.started_at, false)}</span>
          {a.speed_kph != null && <span><Gauge size={12} /> {formatUnit(a.speed_kph, 'km/h')}</span>}
          <span><MapPin size={12} /> {a.segment_name ?? 'Di luar jalur'}</span>
        </div>
      </div>
    </div>
  )
}
