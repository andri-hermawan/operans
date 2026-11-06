// Daftar unit di panel Live Tracking (gaya sistem lama): gambar marker + kode unit; klik satu unit →
// terbuka "Activity Summary" di bawahnya, klik lagi / tombol X untuk menutup.
import { CloseOutlined, InfoCircleFilled } from '@ant-design/icons'
import { Empty } from 'antd'

import type { LiveState } from '@/types/liveState.types'
import type { Activity } from '@/types/telemetry.types'
import { formatNumber, formatTime } from '@/utils/format'
import { unitImageUrl } from '@/components/map/icons'

import styles from './monitoring.module.css'

export type AssetUnit = LiveState & { code: string }

interface Props {
  units: AssetUnit[]
  selectedId?: string
  onSelect: (equipmentId: string) => void
  onClear: () => void
  /** Ringkasan aktivitas per unit (kunci: equipment_id). */
  activity: Map<string, Activity>
  loading?: boolean
  onRefresh?: () => void
}

/** Data yang menentukan gambar marker sebuah unit. */
const markerOf = (u: LiveState) => ({ status: u.equipment_status, vessel: u.vessel_status, alert: u.alert_count > 0, breakdown: !!u.is_breakdown })

export default function AssetList({ units, selectedId, onSelect, onClear, activity, loading, onRefresh }: Props) {
  if (units.length === 0) return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Tidak ada unit" />
  return (
    <div>
      {units.map((u) => {
        const active = u.equipment_id === selectedId
        return (
          <div key={u.equipment_id} className={styles.assetItem}>
            <div className={`${styles.assetRow} ${active ? styles.assetRowActive : ''}`} onClick={() => onSelect(u.equipment_id)}>
              <span className={styles.assetName}>
                <img src={unitImageUrl(markerOf(u))} alt={u.code} className={styles.assetIcon} />
                <b>{u.code}</b>
              </span>
              {active ? (
                <CloseOutlined
                  className={styles.assetAction}
                  title="Tutup"
                  onClick={(e) => {
                    e.stopPropagation()
                    onClear()
                  }}
                />
              ) : (
                <InfoCircleFilled className={styles.assetInfo} />
              )}
            </div>
            {active && (
              <div className={styles.assetDetail}>
                <div className={styles.assetDetailHead}>
                  <b>
                    Activity Summary
                    {loading && <span className={styles.assetLoading}>memuat…</span>}
                  </b>
                  <span className={styles.assetTime} title="Klik untuk memuat ulang" onClick={() => onRefresh?.()}>
                    {u.event_time ? formatTime(u.event_time, false) : '-'}
                  </span>
                </div>
                <ActivityRows a={activity.get(u.equipment_id)} />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function ActivityRows({ a }: { a?: Activity }) {
  const n = (v: number | null | undefined, unit: string, d = 2) => (v == null ? '-' : `${formatNumber(v, d)} ${unit}`)
  const row = (label: string, value: string, sub = false) => (
    <div className={`${styles.actRow} ${sub ? styles.actSub : ''}`}>
      <span className={styles.actLabel}>{label}</span>
      <span className={styles.actValue}>{value}</span>
    </div>
  )
  return (
    <div>
      {row('Running Time', n(a?.running_hours, 'hours'))}
      {row('Empty', n(a?.running_empty_hours, 'hours'), true)}
      {row('Loaded', n(a?.running_loaded_hours, 'hours'), true)}
      {row('Idling Time', n(a?.idle_hours, 'hours'))}
      {row('Empty', n(a?.idle_empty_hours, 'hours'), true)}
      {row('Loaded', n(a?.idle_loaded_hours, 'hours'), true)}
      {row('Avg Speed', n(a?.avg_speed, 'km/h'))}
      {row('Empty', n(a?.avg_speed_empty, 'km/h'), true)}
      {row('Loaded', n(a?.avg_speed_loaded, 'km/h'), true)}
      {row('Max Speed', n(a?.max_speed, 'km/h'))}
      {row('Empty', n(a?.max_speed_empty, 'km/h'), true)}
      {row('Loaded', n(a?.max_speed_loaded, 'km/h'), true)}
      {row('Mileage', n(a?.mileage_km, 'km'))}
      {row('Fuel', '')}
      {row('Start', n(a?.fuel_start_liter, 'L'), true)}
      {row('Remaining', n(a?.fuel_remaining_liter, 'L'), true)}
      {row('Increase', n(a?.fuel_increase_liter, 'L'), true)}
      {row('Decrease', n(a?.fuel_decrease_liter, 'L'), true)}
      {row('Burn Ratio', n(a?.fuel_burn_km_per_liter, 'km/L'), true)}
    </div>
  )
}
