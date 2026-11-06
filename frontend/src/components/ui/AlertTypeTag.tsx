import { alertTypeColor, alertTypeLabel } from '@/utils/colors'

/** Nama jenis alert: latar warna alert yang tipis + teks gelap (selalu terbaca). Warna dari utils/colors. */
export default function AlertTypeTag({ type }: { type?: string | null }) {
  if (!type) return <>-</>
  const c = alertTypeColor[type] ?? '#8A8A8A'
  return (
    <span style={{ display: 'inline-block', padding: '2px 10px', borderRadius: 6, background: `${c}26`, color: '#1f2937', fontWeight: 600, whiteSpace: 'nowrap' }}>
      {alertTypeLabel[type] ?? type}
    </span>
  )
}
