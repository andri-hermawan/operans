// Ikon penanda unit (dipakai Live Tracking, Geofencing, riwayat) — memakai gambar di assets/markers,
// seperti sistem lama (utils/marker-icon.ts). Gambar dipilih dari: status × muatan × (alert, offline, breakdown).
import L from 'leaflet'

import styles from './map.module.css'

// Semua gambar dimuat sekali. Nama berkas dinormalkan (huruf kecil, tanpa spasi) karena ada yang tidak seragam
// ("Running_Loaded _Alert.png").
const files = import.meta.glob('/src/assets/markers/*.png', { eager: true, query: '?url', import: 'default' }) as Record<string, string>
const IMAGES = new Map(
  Object.entries(files).map(([path, url]) => [path.split('/').pop()!.replace(/\.png$/i, '').replace(/\s/g, '').toLowerCase(), url]),
)

type BaseStatus = 'running' | 'idle' | 'stop'

/** Nama berkas: <status>_<muatan>[_<kombinasi>]. Dua tanda → "2Combine", tiga → "3Combine". */
function imageUrl(status: BaseStatus, vessel: string, flags: { alert: boolean; offline: boolean; breakdown: boolean }) {
  const parts = [flags.alert && 'alert', flags.offline && 'offline', flags.breakdown && 'breakdown'].filter(Boolean) as string[]
  const combine = parts.length > 1 ? `_${parts.length}combine` : ''
  const key = `${status}_${vessel}${combine}${parts.map((p) => `_${p}`).join('')}`.toLowerCase()
  return IMAGES.get(key)
}

/** Unit tanpa status yang dikenal (offline / belum ada data) memakai dasar "stop" + tanda offline, seperti sistem lama. */
const baseOf = (status?: string | null): BaseStatus => (status === 'running' || status === 'idle' ? status : 'stop')

type UnitIconOptions = { code: string; status?: string | null; vessel?: string | null; heading?: number | null; alert?: boolean; breakdown?: boolean; selected?: boolean; showLabel?: boolean }

/** Alamat gambar marker sebuah unit (dipakai juga oleh daftar unit di panel). */
export function unitImageUrl(o: Pick<UnitIconOptions, 'status' | 'vessel' | 'alert' | 'breakdown'>) {
  const vessel = o.vessel === 'loaded' || o.vessel === 'empty' ? o.vessel : 'unknown'
  const offline = !o.status || o.status === 'offline'
  return imageUrl(baseOf(o.status), vessel, { alert: !!o.alert, offline, breakdown: !!o.breakdown })
}

export function unitIcon(o: UnitIconOptions) {
  const url = unitImageUrl(o)
  const size = o.selected ? 34 : 28
  const img = url ? `<img src="${url}" width="${size}" height="${size}" alt="" draggable="false" />` : ''
  const label = o.showLabel ? `<span class="${styles.unitLabel}">${o.code}</span>` : ''
  return L.divIcon({
    className: styles.unitIconWrap,
    html: `<div class="${styles.unit} ${o.selected ? styles.selected : ''}" style="width:${size}px;height:${size}px">${img}</div>${label}`,
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
    popupAnchor: [0, -size / 2],
  })
}
