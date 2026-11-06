// Lonceng notifikasi alert di header. Alert masuk langsung dari WebSocket; membuka lonceng menampilkan semua alert yang
// belum dibaca, dan menutup laci (tombol X) MENGHAPUS seluruh daftar — yang muncul berikutnya hanya alert baru.
import { useState } from 'react'
import { Badge, Button, Drawer, Empty, Spin, Typography } from 'antd'
import { Bell } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { useAlertBell } from '@/hooks/useAlertBell'
import { formatNumber, formatRelative, formatUnit } from '@/utils/format'
import AlertTypeTag from '@/components/ui/AlertTypeTag'
import styles from '@/layouts/layout.module.css'

type BellAlert = ReturnType<typeof useAlertBell>['alerts'][number]

/** Keterangan singkat di kanan: kecepatan (speed), selisih liter (fuel), atau nama jalur (off track). */
const detailOf = (a: BellAlert): string => {
  if (a.alert_type.startsWith('fuel')) return a.fuel_delta_liter == null ? '-' : `${formatNumber(a.fuel_delta_liter, 1)} L`
  if (a.alert_type.endsWith('speed')) return a.speed_kph == null ? '-' : formatUnit(a.speed_kph, 'km/h')
  return a.segment_name ?? '-'
}

export default function AlertBell() {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()
  const { alerts, unread, date, loading, clearAll } = useAlertBell()

  // X / klik di luar laci / Esc → hapus seluruh daftar
  const closeAndClear = () => {
    setOpen(false)
    clearAll()
  }
  // membuka satu alert atau "lihat semua": pindah halaman tanpa menghapus alert yang lain
  const goHistory = (a: BellAlert) => {
    setOpen(false)
    const p = new URLSearchParams({ equipment: a.equipment_id })
    if (a.operational_date) p.set('date', a.operational_date)
    if (a.shift_code) p.set('shift', a.shift_code)
    navigate(`/position-history?${p.toString()}`)
  }
  const goAll = () => {
    setOpen(false)
    navigate(`/alert?from=${date}&to=${date}`)
  }

  return (
    <>
      <Badge count={unread} size="small" overflowCount={99}>
        <Button
          type="text"
          shape="circle"
          aria-label="Notifikasi alert"
          className={`${styles.bell} ${unread > 0 ? styles.bellUnread : ''}`}
          icon={<Bell size={19} />}
          onClick={() => setOpen(true)}
        />
      </Badge>

      <Drawer
        title="Notifikasi alert"
        open={open}
        onClose={closeAndClear}
        placement="right"
        size="default"
        styles={{ body: { padding: 0 } }}
        footer={
          <div style={{ textAlign: 'center' }}>
            <Button type="link" onClick={goAll}>Lihat semua alert</Button>
          </div>
        }
      >
        <div style={{ padding: '8px 20px', borderBottom: '1px solid #f0f0f0' }}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {unread > 0 ? `${unread} belum dibaca · menutup notifikasi akan menghapus daftar ini` : 'Alert baru akan muncul di sini secara langsung'}
          </Typography.Text>
        </div>
        {loading && alerts.length === 0 ? (
          <div style={{ padding: 48, textAlign: 'center' }}><Spin /></div>
        ) : alerts.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Tidak ada notifikasi" style={{ padding: 48 }} />
        ) : (
          alerts.map((a) => (
            <button key={a.id} type="button" className={styles.alertItem} onClick={() => goHistory(a)}>
              <div className={styles.alertRow}>
                <AlertTypeTag type={a.alert_type} />
                <span className={styles.alertMuted}>{formatRelative(a.started_at)}</span>
              </div>
              <div className={styles.alertRow}>
                <span style={{ fontSize: 13 }}>
                  <b>{a.code}</b>
                  {a.segment_name && <span className={styles.alertMuted}> · {a.segment_name}</span>}
                </span>
                <span className={styles.alertMuted}>{detailOf(a)}</span>
              </div>
            </button>
          ))
        )}
      </Drawer>
    </>
  )
}
