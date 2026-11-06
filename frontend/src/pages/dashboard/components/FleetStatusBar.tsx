// Kondisi armada saat ini (live): satu batang bertingkat + jumlah tiap status. Mudah dibaca sekilas: hijau = bekerja.
import { Empty } from 'antd'

import { equipmentStatusColor, equipmentStatusLabel } from '@/utils/colors'

import styles from '../dashboard.module.css'

const ORDER = ['running', 'idle', 'stop', 'breakdown', 'offline']

export default function FleetStatusBar({ byStatus, total }: { byStatus: Record<string, number>; total: number }) {
  if (!total) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Belum ada data unit" />
  return (
    <div>
      <div className={styles.statusBar}>
        {ORDER.filter((k) => byStatus[k]).map((k) => (
          <span key={k} title={`${equipmentStatusLabel[k]}: ${byStatus[k]}`} style={{ width: `${(byStatus[k] / total) * 100}%`, background: equipmentStatusColor[k] }} />
        ))}
      </div>
      <div className={styles.statusLegend}>
        {ORDER.map((k) => (
          <div key={k} className={styles.statusItem}>
            <span className={styles.statusDot} style={{ background: equipmentStatusColor[k] }} />
            <div>
              <div className={styles.statusCount}>{byStatus[k] ?? 0}</div>
              <div className={styles.statusName}>{equipmentStatusLabel[k]}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
