// Halaman cetak Alert Summary (tanpa menu). Dipakai:
//  1) core-api: browser tanpa layar membuka /print/report-alert-summary?date&shift#token=… lalu menyimpan PDF
//     (token di bagian "#" tidak terkirim ke server/log; langsung dihapus dari alamat);
//  2) pengguna: tombol "Cetak / PDF" di laporan (memakai sesi login biasa).
// Token sementara dipasang di auth/printToken.ts (main.tsx).
// Saat data & grafik siap → window.__REPORT_READY__ = true (tanda bagi pembuat PDF).
import { useEffect } from 'react'
import { Button, Result, Spin } from 'antd'
import { Printer } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'

import { errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import { useAlerts } from '@/hooks/useAlerts'
import { useShifts } from '@/hooks/useShifts'
import { useZones } from '@/hooks/useZones'
import { formatDateTime, formatPlainDate } from '@/utils/format'
import AlertSummaryReport from '@/components/report/AlertSummaryReport'
import logo from '@/assets/logo.png'

import styles from './print.module.css'

export default function PrintAlertSummary() {
  const [params] = useSearchParams()
  const date = params.get('date') ?? undefined
  const shiftCode = params.get('shift') ?? undefined
  const signedIn = useAuthStore((s) => s.status === 'signedIn')
  const project = useAuthStore((s) => s.project)
  const shifts = useShifts()
  const alerts = useAlerts({ date, shift: shiftCode })
  const zones = useZones() // grafik lokasi kejadian memakai daftar zona: halaman baru "siap" setelah zona dimuat
  const shift = shifts.data?.find((s) => s.shift_code === shiftCode)

  useEffect(() => {
    if (alerts.error || shifts.error || zones.error) window.__REPORT_ERROR__ = errorMessage(alerts.error ?? shifts.error ?? zones.error)
    if (alerts.isSuccess && shifts.isSuccess && zones.isSuccess) {
      const t = setTimeout(() => (window.__REPORT_READY__ = true), 1200) // beri waktu grafik tergambar
      return () => clearTimeout(t)
    }
  }, [alerts.isSuccess, shifts.isSuccess, zones.isSuccess, alerts.error, shifts.error, zones.error])

  if (!signedIn) return <Result status="403" title="Perlu login" subTitle="Buka laporan ini dari menu Report setelah login." />
  if (!date) return <Result status="warning" title="Tanggal belum dipilih" />

  return (
    <div className={`${styles.page} ${params.get('preview') ? '' : styles.pdfWidth}`}>
      {/* cetak manual dari browser (preview=1): A4 mendatar. PDF dari core-api TIDAK memakai ini: ukuran kertasnya ditentukan
          core-api (satu halaman, tinggi mengikuti isi) dan aturan @page di sini akan memutar kertasnya. */}
      {params.get('preview') && <style>{'@page { size: A4 landscape; margin: 8mm }'}</style>}
      <header className={styles.head}>
        <img src={logo} alt="RMKO" height={40} />
        <div style={{ flex: 1 }}>
          <div className={styles.title}>Alert Summary</div>
          <div className={styles.sub}>
            {project?.project_code ? `${project.project_code} · ` : ''}Tanggal operasional {formatPlainDate(date)} · {shift ? `${shift.shift_name} (${shift.start_time}–${shift.end_time})` : 'Semua shift'}
          </div>
        </div>
        <div className={styles.sub}>Dibuat {formatDateTime(new Date().toISOString())}</div>
        {params.get('preview') && <Button className={styles.noPrint} icon={<Printer size={14} />} onClick={() => window.print()}>Cetak / Simpan PDF</Button>}
      </header>
      {alerts.isLoading || shifts.isLoading ? <Spin style={{ display: 'block', margin: 60 }} /> : (
        <AlertSummaryReport alerts={alerts.data?.data ?? []} shift={shift} shifts={shifts.data ?? []} animate={false} fixedLayout />
      )}
    </div>
  )
}
