// Asset Passing (gaya sistem lama): daftar kejadian masuk/keluar area — kode unit & area, jam & IN/OUT; footer Total + Last.
import { Empty, Tag } from 'antd'

import type { GeofenceEvent } from '@/types/geofence.types'
import { formatTime } from '@/utils/format'
import styles from '@/components/monitoring/monitoring.module.css'
import SectionCard from '@/components/ui/SectionCard'

export type PassingRow = GeofenceEvent & { code: string }

export default function AssetPassing({ rows, emptyText = 'Belum ada kejadian' }: { rows: PassingRow[]; emptyText?: string }) {
  const last = rows.length ? rows.reduce((a, b) => (a.event_time > b.event_time ? a : b)).event_time : undefined
  return (
    <SectionCard title="Asset Passing" className={`${styles.fillCard} ${styles.fillCol}`} styles={{ body: { padding: 0 } }}>
      <div className={styles.scrollBox}>
        {rows.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={emptyText} />}
        {rows.map((r) => (
          <div key={r.id} className={styles.passRow}>
            <div className={styles.passTop}>
              <span>{r.code}</span>
              <span>{r.segment_name || '-'}</span>
            </div>
            <div className={styles.passBottom}>
              <span>{formatTime(r.event_time, false)}</span>
              <Tag color={r.event_type === 'enter' ? 'green' : 'red'} style={{ margin: 0 }}>
                {r.event_type === 'enter' ? 'IN' : 'OUT'}
              </Tag>
            </div>
          </div>
        ))}
      </div>
      <div className={styles.cardFooter}>
        <span>Total : <b>{rows.length}</b></span>
        <span>Last : <b>{last ? formatTime(last, false) : '-'}</b></span>
      </div>
    </SectionCard>
  )
}
