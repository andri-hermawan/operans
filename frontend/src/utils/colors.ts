// Palet warna aplikasi. Komponen TIDAK menulis kode warna sendiri: ambil dari sini (atau token tema antd).

/** Warna merek & dasar (dipakai juga oleh tema antd di app/theme.ts). */
export const brand = {
  primary: '#0b4f8c', // biru RMKO, lebih lembut dari #064596 lama
  primaryDark: '#083a68',
  accent: '#f0a500', // aksen kuning tambang
  success: '#15803d',
  warning: '#d97706',
  danger: '#dc2626',
  info: '#0284c7',
  muted: '#64748b',
  border: '#e5e7eb',
  surface: '#ffffff',
  background: '#f4f6f9',
  text: '#1f2937',
} as const

/** Status operasi unit (kode baku ADR 0004). */
export const equipmentStatusColor: Record<string, string> = {
  running: '#16a34a',
  idle: '#f59e0b',
  stop: '#dc2626',
  offline: '#94a3b8',
  breakdown: '#7c3aed',
}

/** Label status seperti sistem lama. */
export const equipmentStatusLabel: Record<string, string> = {
  running: 'Running',
  idle: 'Idle',
  stop: 'Stop',
  offline: 'Offline',
  breakdown: 'Breakdown',
}
export const vesselStatusLabel: Record<string, string> = { loaded: 'Loaded', empty: 'Empty', unknown: 'Unknown' }

/** Status muatan. */
export const vesselStatusColor: Record<string, string> = {
  loaded: '#0b4f8c',
  empty: '#0ea5e9',
  unknown: '#94a3b8',
}

/** Jenis alert (kode baku iot.alerts.alert_type). Palet mengikuti sistem lama. */
export const alertTypeColor: Record<string, string> = {
  overspeed: '#C87033',
  underspeed: '#DCA62D',
  off_track: '#8A8A8A',
  fuel_decrease_engine_on: '#5186B3',
  fuel_decrease_engine_off: '#5186B3',
}

/** Label alert seperti sistem lama. */
export const alertTypeLabel: Record<string, string> = {
  overspeed: 'Overspeed',
  underspeed: 'Underspeed',
  off_track: 'Off Track',
  fuel_decrease_engine_on: 'Fuel Decrease Engine On',
  fuel_decrease_engine_off: 'Fuel Decrease Engine Off',
}

/** Pita kecepatan (speed heatmap) — batas atas inklusif, sama dengan sistem lama. */
export const speedBands: { max: number; color: string; label: string }[] = [
  { max: 10, color: '#111827', label: '0–10' },
  { max: 20, color: '#f97316', label: '11–20' },
  { max: 30, color: '#16a34a', label: '21–30' },
  { max: 40, color: '#06b6d4', label: '31–40' },
  { max: 50, color: '#2563eb', label: '41–50' },
  { max: Infinity, color: '#dc2626', label: '> 50' },
]
export const speedColor = (kph: number): string => (speedBands.find((b) => kph <= b.max) ?? speedBands[speedBands.length - 1]).color

/** Skala laporan bergaya Excel (merah/kuning/hijau). */
export const reportScale = {
  bad: { background: '#fee2e2', color: '#991b1b' },
  warn: { background: '#fef3c7', color: '#92400e' },
  good: { background: '#dcfce7', color: '#166534' },
} as const

/** Warna teks yang terbaca di atas warna latar (hitam/putih menurut kecerahan). */
export const readableText = (hex: string): string => {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(full.slice(i, i + 2), 16))
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.6 ? '#111827' : '#ffffff'
}

/** Warna untuk kode apa pun dengan cadangan abu-abu. */
export const colorOf = (map: Record<string, string>, code?: string | null): string =>
  (code && map[code.toLowerCase()]) || '#94a3b8'
