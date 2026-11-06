import { Tag } from 'antd'

import { colorOf } from '@/utils/colors'

interface Props {
  /** Kode nilai (running, loaded, overspeed, ...). */
  value?: string | null
  /** Peta warna dari lib/colors (equipmentStatusColor, alertTypeColor, ...). */
  colors: Record<string, string>
  /** Peta label tampilan; tanpa ini kode ditampilkan dengan huruf awal besar. */
  labels?: Record<string, string>
}

const titleCase = (s: string) => s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())

/** Label status/kategori: latar warna tipis + titik berwarna + teks gelap (selalu terbaca). Warna & label dari utils/colors. */
export default function StatusTag({ value, colors, labels }: Props) {
  if (!value) return <>-</>
  const c = colorOf(colors, value)
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, padding: '2px 10px', borderRadius: 6, background: `${c}26`, color: '#1f2937', fontWeight: 600, whiteSpace: 'nowrap' }}>
      <span style={{ width: 8, height: 8, borderRadius: '50%', background: c }} />
      {labels?.[value.toLowerCase()] ?? titleCase(value)}
    </span>
  )
}

/** Aktif / Nonaktif. */
export const ActiveTag = ({ active }: { active: boolean }) => (
  <Tag color={active ? 'success' : 'default'} style={{ marginInlineEnd: 0 }}>{active ? 'Active' : 'Inactive'}</Tag>
)
