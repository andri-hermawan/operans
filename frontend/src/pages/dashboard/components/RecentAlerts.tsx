// Alert terbaru pada shift ini (paling baru di atas).
import { Empty } from 'antd'

import type { AlertRow } from '@/types/alert.types'
import { formatTime, formatUnit } from '@/utils/format'
import AlertTypeTag from '@/components/ui/AlertTypeTag'

import styles from '../dashboard.module.css'

export default function RecentAlerts({ items }: { items: (AlertRow & { code: string })[] }) {
  if (!items.length) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Tidak ada alert" />
  return (
    <div className={styles.feed}>
      {items.map((a) => (
        <div key={a.id} className={styles.feedItem}>
          <div className={styles.feedBody}>
            <div className={styles.feedTop}>
              <b>{a.code}</b>
              <span className={styles.feedMeta}>{formatTime(a.started_at, false)}</span>
            </div>
            <div className={styles.feedTop} style={{ marginTop: 4 }}>
              <AlertTypeTag type={a.alert_type} />
              <span className={styles.feedMeta}>{a.speed_kph != null ? formatUnit(a.speed_kph, 'km/h') : ''}{a.segment_name ? ` · ${a.segment_name}` : ''}</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
