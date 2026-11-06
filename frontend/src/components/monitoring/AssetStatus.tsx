// Ringkasan status unit di panel Live Tracking (gaya sistem lama): unit dikelompokkan menurut gambar marker-nya
// (status × muatan × alert/offline/breakdown). Klik satu kelompok → tampil jam data terakhir + daftar kode unit.
import { useMemo, useState } from 'react'
import { MenuOutlined, ReloadOutlined } from '@ant-design/icons'
import { Empty } from 'antd'

import { formatTime } from '@/utils/format'
import { unitImageUrl } from '@/components/map/icons'

import type { AssetUnit } from './AssetList'
import styles from './monitoring.module.css'

interface Group {
  url: string
  codes: string[]
  lastUpdate?: string
}

/** `units` = unit yang diringkas (satu unit bila ada yang dipilih, selain itu semuanya). */
export default function AssetStatus({ units }: { units: AssetUnit[] }) {
  const [open, setOpen] = useState<string>()

  const groups = useMemo(() => {
    const m = new Map<string, Group>()
    for (const u of units) {
      const url = unitImageUrl({ status: u.equipment_status, vessel: u.vessel_status, alert: u.alert_count > 0, breakdown: !!u.is_breakdown }) ?? ''
      const g = m.get(url) ?? { url, codes: [] }
      g.codes.push(u.code)
      if (u.event_time && (!g.lastUpdate || u.event_time > g.lastUpdate)) g.lastUpdate = u.event_time
      m.set(url, g)
    }
    return [...m.values()].sort((a, b) => b.codes.length - a.codes.length)
  }, [units])

  return (
    <div>
      <div className={styles.statusTotal}>
        <ReloadOutlined style={{ fontSize: 22 }} />
        <b>{units.length} Total</b>
      </div>
      {groups.length === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Belum ada status unit" />}
      {groups.map((g) => {
        const expanded = open === g.url
        return (
          <div key={g.url} className={styles.assetItem}>
            <div className={`${styles.assetRow} ${expanded ? styles.assetRowActive : ''}`} onClick={() => setOpen(expanded ? undefined : g.url)}>
              <span className={styles.assetName} style={{ fontStyle: 'normal' }}>
                {g.url && <img src={g.url} alt="" className={styles.assetIcon} />}
                <b>{g.codes.length} Dump Truck</b>
              </span>
              <MenuOutlined className={styles.assetAction} style={{ fontSize: 14 }} />
            </div>
            {expanded && (
              <div className={styles.statusDetail}>
                <span className={styles.statusUpdate}>Last Update {g.lastUpdate ? formatTime(g.lastUpdate) : '-'}</span>
                {[...g.codes].sort().map((c) => (
                  <div key={c}>- {c}</div>
                ))}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
