// Satu "bab" Dashboard: nomor + judul pendek, deret kartu angka berikon, lalu isi (grafik / daftar).
// Urutan bab-bab itulah ceritanya: kondisi armada → produksi → waktu kerja → breakdown → pergerakan → alert → rincian.
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

import { brand } from '@/utils/colors'

import styles from '../dashboard.module.css'

export interface Stat {
  label: string
  value: ReactNode
  unit?: string
  icon: LucideIcon
  /** warna ikon + angka */
  color?: string
}

interface Props {
  no: number
  title: string
  /** Warna nomor bab: merah/kuning/hijau menurut keadaan, biru bila netral. */
  tone?: 'neutral' | 'good' | 'warn' | 'bad'
  stats?: Stat[]
  children?: ReactNode
}

const TONE = { neutral: brand.primary, good: brand.success, warn: brand.warning, bad: brand.danger }

export default function Chapter({ no, title, tone = 'neutral', stats, children }: Props) {
  return (
    <section className={styles.chapter}>
      <header className={styles.chapterHead}>
        <span className={styles.chapterNo} style={{ background: TONE[tone] }}>{no}</span>
        <h3 className={styles.chapterTitle}>{title}</h3>
      </header>
      {stats && stats.length > 0 && (
        <div className={styles.statGrid}>
          {stats.map(({ icon: Icon, ...s }) => {
            const color = s.color ?? brand.primary
            return (
              <div key={s.label} className={styles.stat}>
                <span className={styles.statIcon} style={{ background: `${color}1f`, color }}>
                  <Icon size={20} />
                </span>
                <div>
                  <div className={styles.statValue} style={s.color ? { color } : undefined}>
                    {s.value}
                    {s.unit && <span className={styles.statUnit}>{s.unit}</span>}
                  </div>
                  <div className={styles.statLabel}>{s.label}</div>
                </div>
              </div>
            )
          })}
        </div>
      )}
      {children}
    </section>
  )
}
