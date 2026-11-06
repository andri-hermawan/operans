import type { ReactNode } from 'react'

import type { InfoData } from '@/types/monitoring.types'
import { equipmentStatusColor, equipmentStatusLabel, vesselStatusColor, vesselStatusLabel } from '@/utils/colors'
import { formatNumber, formatPercent, formatTime, formatUnit } from '@/utils/format'
import { googleMapsUrl } from '@/utils/map'
import StatusTag from '@/components/ui/StatusTag'

import styles from './monitoring.module.css'

/** Isi popup titik/unit — satu bentuk untuk semua peta (pengganti 5 salinan di sistem lama). */
export default function InfoRows({ title, d }: { title: ReactNode; d: InfoData }) {
  const rows: [string, ReactNode][] = [
    ['Waktu', d.time ? formatTime(d.time) : undefined],
    ['Operator', d.operator],
    ['Status', d.status ? <StatusTag value={d.status} colors={equipmentStatusColor} labels={equipmentStatusLabel} /> : undefined],
    ['Muatan', d.vessel ? <StatusTag value={d.vessel} colors={vesselStatusColor} labels={vesselStatusLabel} /> : undefined],
    ['Mesin', d.engine == null ? undefined : d.engine ? 'ON' : 'OFF'],
    ['Speed', d.speed == null ? undefined : formatUnit(d.speed, 'km/h')],
    ['Fuel', d.fuelLiter == null ? undefined : `${formatNumber(d.fuelLiter, 1)} L · ${formatPercent(d.fuelPercent)}`],
    ['Segment', d.segment ?? undefined],
    ...(d.extra ?? []).map((e) => [e.label, e.value] as [string, ReactNode]),
  ]
  const url = googleMapsUrl(d.lat, d.lon)
  return (
    <div className={styles.info}>
      <div className={styles.infoTitle}>{title}</div>
      {rows.filter(([, v]) => v !== undefined && v !== null && v !== '').map(([k, v]) => (
        <div key={k} className={styles.infoRow}>
          <span>{k}</span>
          <span>{v}</span>
        </div>
      ))}
      {url && <a href={url} target="_blank" rel="noreferrer">Buka di Google Maps</a>}
    </div>
  )
}

/** Baris judul daftar bergaya FMS lama ("Abnormal Alert List · 12 Total"). */
