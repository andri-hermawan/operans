import type { ReactNode } from 'react'
import { Typography } from 'antd'

import styles from './monitoring.module.css'

/** Judul daftar di panel kanan halaman pemantauan. */
const ListHeader = ({ title, count, extra }: { title: string; count?: number; extra?: ReactNode }) => (
  <div className={styles.listHeader}>
    <span>{title}{count != null && <Typography.Text style={{ color: '#fff', opacity: 0.85 }}> · {count} Total</Typography.Text>}</span>
    {extra}
  </div>
)

export default ListHeader
