// Kondisi armada saat ini per tipe unit (Dump Truck, Water Truck, Fuel Truck): jumlah Running, Idle, Stop, Breakdown.
import { Empty } from 'antd'
import { Droplets, Fuel, Truck } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import type { FleetTypeRow } from '@/hooks/useDashboardData'
import { equipmentStatusColor, equipmentStatusLabel } from '@/utils/colors'

import styles from '../dashboard.module.css'

const STATUSES = ['running', 'idle', 'stop', 'breakdown'] as const
const ICON: Record<string, LucideIcon> = { 'Dump Truck': Truck, 'Water Truck': Droplets, 'Fuel Truck': Fuel }

export default function FleetByType({ rows }: { rows: FleetTypeRow[] }) {
  if (!rows.some((r) => r.total)) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Belum ada data unit" />
  return (
    <div className={styles.typeGrid}>
      {rows.map((r) => {
        const Icon = ICON[r.type] ?? Truck
        return (
          <div key={r.type} className={styles.typeCard}>
            <div className={styles.typeHead}>
              <span className={styles.typeIcon}><Icon size={18} /></span>
              <span className={styles.typeName}>{r.type}</span>
              <span className={styles.typeTotal}>{r.total} unit · {r.online} online</span>
            </div>
            <div className={styles.statusBar}>
              {STATUSES.filter((k) => r[k]).map((k) => (
                <span key={k} title={`${equipmentStatusLabel[k]}: ${r[k]}`} style={{ width: `${(r[k] / r.total) * 100}%`, background: equipmentStatusColor[k] }} />
              ))}
            </div>
            <div className={styles.typeStatuses}>
              {STATUSES.map((k) => (
                <div key={k} className={styles.statusItem}>
                  <span className={styles.statusDot} style={{ background: equipmentStatusColor[k] }} />
                  <div>
                    <div className={styles.statusCount}>{r[k]}</div>
                    <div className={styles.statusName}>{equipmentStatusLabel[k]}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
