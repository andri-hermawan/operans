// Kerangka halaman cetak laporan (tanpa menu): kop (logo, judul, project, tanggal) + isi. Dipakai halaman cetak Executive Summary;
// alamat & tanda siap (window.__REPORT_READY__) sama dengan Alert Summary (lihat PrintReportAlertSummaryPage).
import { useEffect, type ReactNode } from 'react'
import { Button, Result, Spin } from 'antd'
import { Printer } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'

import { useAuthStore } from '@/stores/auth.store'
import { formatDateTime } from '@/utils/format'
import logo from '@/assets/logo.png'

import styles from './print.module.css'

interface Props {
  title: string
  /** Tanggal laporan yang ditampilkan (parameter `date` dari alamat; laporan Executive menampilkan data sampai H-1-nya). */
  subtitle: (date: string) => string
  loading: boolean
  /** Data sudah siap tampil → pembuat PDF boleh mencetak. */
  ready: boolean
  error?: string
  children: (date: string) => ReactNode
}

export default function PrintFrame({ title, subtitle, loading, ready, error, children }: Props) {
  const [params] = useSearchParams()
  const date = params.get('date') ?? undefined
  const signedIn = useAuthStore((s) => s.status === 'signedIn')
  const project = useAuthStore((s) => s.project)

  useEffect(() => {
    if (error) window.__REPORT_ERROR__ = error
    if (ready) {
      const t = setTimeout(() => (window.__REPORT_READY__ = true), 600)
      return () => clearTimeout(t)
    }
  }, [ready, error])

  if (!signedIn) return <Result status="403" title="Perlu login" subTitle="Buka laporan ini dari menu Report setelah login." />
  if (!date) return <Result status="warning" title="Tanggal belum dipilih" />

  return (
    <div className={`${styles.page} ${params.get('preview') ? '' : styles.pdfWidth}`}>
      {params.get('preview') && <style>{'@page { size: A4 landscape; margin: 8mm }'}</style>}
      <header className={styles.head}>
        <img src={logo} alt="RMKO" height={40} />
        <div style={{ flex: 1 }}>
          <div className={styles.title}>{title}</div>
          <div className={styles.sub}>
            {project?.project_code ? `${project.project_code} · ` : ''}
            {subtitle(date)}
          </div>
        </div>
        <div className={styles.sub}>Dibuat {formatDateTime(new Date().toISOString())}</div>
        {params.get('preview') && <Button className={styles.noPrint} icon={<Printer size={14} />} onClick={() => window.print()}>Cetak / Simpan PDF</Button>}
      </header>
      {loading ? <Spin style={{ display: 'block', margin: 60 }} /> : children(date)}
    </div>
  )
}
