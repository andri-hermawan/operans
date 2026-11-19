// Durasi antrian hauling di atas daftar Position History: dari unit berhenti (speed 0) di Km 01+000 - 02+000
// sampai keluar dari Stockpile Emplacement / TLS2 Zona 4. Satu baris per ritase. Aturan: utils/haulCycle.ts.
import { useMemo, useRef, useState } from 'react'
import { message } from 'antd'
import { ArrowRight, Camera, Hourglass, Loader2, Truck } from 'lucide-react'

import type { Point } from '@/types/telemetry.types'
import { useShifts } from '@/hooks/useShifts'
import { captureNodeToPng, downloadBlob } from '@/utils/captureNode'
import { findHaulCycles, formatSeconds } from '@/utils/haulCycle'
import { formatPlainDate, formatTime } from '@/utils/format'

import styles from './haul.module.css'

interface Props {
  rows: Point[]
  /** kode unit (mis. DT10250), tanggal operasional (YYYY-MM-DD) dan kode shift — tampil di kartu & nama berkas */
  code: string
  date?: string
  shiftCode?: string
  onSelect: (index: number) => void
  loading?: boolean
}

export default function HaulCycleSummary({ rows, code, date, shiftCode, onSelect, loading }: Props) {
  const cycles = useMemo(() => findHaulCycles(rows), [rows])
  const { data: shifts = [] } = useShifts()
  const shiftName = shiftCode ? (shifts.find((s) => s.shift_code === shiftCode)?.shift_name ?? shiftCode) : 'Semua shift'
  const ref = useRef<HTMLElement>(null)
  const [busy, setBusy] = useState(false)

  const capture = async () => {
    if (!ref.current || busy) return
    setBusy(true)
    try {
      const blob = await captureNodeToPng(ref.current)
      downloadBlob(blob, `antrian-hauling_${code}_${date ?? 'tanggal'}_${shiftCode ?? 'semua-shift'}.png`)
      message.success('Gambar disimpan')
    } catch {
      message.error('Gagal menangkap gambar')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section ref={ref} className={styles.card} aria-label="Durasi antrian hauling">
      <header className={styles.head}>
        <span className={styles.headIcon}>
          <Hourglass size={18} />
        </span>
        <div className={styles.headText}>
          <b>Durasi Antrian Hauling</b>
          <span className={styles.route}>
            Km 01+000 - 02+000 <ArrowRight size={11} /> Stockpile / TLS2 Zona 4
          </span>
        </div>
        <button type="button" className={styles.capture} onClick={capture} disabled={busy || cycles.length === 0} data-capture-ignore aria-label="Capture gambar" title="Capture gambar">
          {busy ? <Loader2 size={15} className={styles.spin} /> : <Camera size={15} />}
          <span>Capture</span>
        </button>
      </header>

      <div className={styles.meta}>
        <span><small>Asset ID</small><b>{code}</b></span>
        <span><small>Tanggal</small><b>{formatPlainDate(date)}</b></span>
        <span><small>Shift</small><b>{shiftName}</b></span>
      </div>

      {cycles.length === 0 ? (
        <div className={styles.empty}>
          <Truck size={22} />
          <span>{loading ? 'Memuat data…' : 'Belum ada ritase pada shift ini.'}</span>
        </div>
      ) : (
        <ul className={styles.list}>
          {cycles.map((c, n) => (
            <li key={c.startIdx}>
              <button type="button" className={styles.item} onClick={() => onSelect(c.startIdx)}>
                <span className={styles.no}>{n + 1}</span>
                <span className={styles.rowBody}>
                  <span className={styles.rowLabel}>Ritase {n + 1}{c.ongoing && <span className={styles.badgeLive}>masih di zona</span>}</span>
                  <span className={styles.time}>
                    {formatTime(c.start)} <ArrowRight size={11} /> {formatTime(c.end)}
                  </span>
                </span>
                <b className={styles.total}>{formatSeconds(c.seconds)}</b>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
