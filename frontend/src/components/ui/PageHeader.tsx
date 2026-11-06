import type { ReactNode } from 'react'
import { Typography } from 'antd'

import styles from '@/components/components.module.css'

interface Props {
  title: ReactNode
  subtitle?: ReactNode
  /** Tombol di kanan (Filter, Actions, Refresh, ...). Turun ke bawah judul di HP. */
  extra?: ReactNode
}

export default function PageHeader({ title, subtitle, extra }: Props) {
  return (
    <div className={styles.pageHeader}>
      <div>
        <Typography.Title level={4} className={styles.pageTitle}>{title}</Typography.Title>
        {subtitle && <Typography.Text type="secondary">{subtitle}</Typography.Text>}
      </div>
      {extra && <div className={styles.pageExtra}>{extra}</div>}
    </div>
  )
}
