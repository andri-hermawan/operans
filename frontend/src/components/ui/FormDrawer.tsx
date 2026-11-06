import type { ReactNode } from 'react'
import { Button, Drawer, Grid, Space } from 'antd'

interface Props {
  open: boolean
  title: ReactNode
  onClose: () => void
  /** Dipanggil oleh tombol Simpan (biasanya form.submit). */
  onSubmit: () => void
  saving?: boolean
  width?: number
  children: ReactNode
}

/** Laci form tambah/ubah standar: penuh layar di HP, Batal/Simpan di kanan atas. */
export default function FormDrawer({ open, title, onClose, onSubmit, saving, width = 480, children }: Props) {
  const screens = Grid.useBreakpoint()
  return (
    <Drawer
      open={open}
      title={title}
      onClose={onClose}
      size={screens.md === false ? '100%' : width}
      destroyOnHidden
      mask={{ closable: !saving }}
      extra={
        <Space>
          <Button onClick={onClose} disabled={saving}>Batal</Button>
          <Button type="primary" onClick={onSubmit} loading={saving}>Simpan</Button>
        </Space>
      }
    >
      {children}
    </Drawer>
  )
}
