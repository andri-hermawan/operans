// Unit yang sedang / pernah breakdown pada shift ini: kategori, jam, status perbaikan.
import { Empty, Tag } from 'antd'
import { Wrench } from 'lucide-react'

import type { EquipmentBreakdown } from '@/types/daily.types'
import { equipmentStatusColor } from '@/utils/colors'
import { formatNumber, formatTime } from '@/utils/format'

import styles from '../dashboard.module.css'

export default function BreakdownFeed({ items }: { items: EquipmentBreakdown[] }) {
  const list = items.filter((b) => b.status === 'breakdown' || b.status === 'continue')
  if (!list.length) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Tidak ada breakdown" />
  const color = equipmentStatusColor.breakdown
  return (
    <div className={styles.feed}>
      {list.map((b) => (
        <div key={b.id} className={styles.feedItem}>
          <span className={styles.feedBadge} style={{ background: `${color}1f`, color }}>
            <Wrench size={16} />
          </span>
          <div className={styles.feedBody}>
            <div className={styles.feedTop}>
              <b>{b.equipment_code} · {b.category_label}</b>
              <Tag color={b.repair_status_code === 'done' ? 'success' : 'orange'} style={{ margin: 0 }}>{b.repair_status_label}</Tag>
            </div>
            <div className={styles.feedMeta}>
              {formatTime(b.started_at, false)} – {b.ended_at ? formatTime(b.ended_at, false) : 'berjalan'}
              {b.duration_hours != null && ` · ${formatNumber(b.duration_hours, 2)} jam`}
              {b.location && ` · ${b.location}`}
            </div>
            {b.description && <div className={styles.feedMeta}>{b.description}</div>}
          </div>
        </div>
      ))}
    </div>
  )
}
