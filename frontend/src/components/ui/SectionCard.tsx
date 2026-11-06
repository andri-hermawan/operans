import type { ReactNode } from 'react'
import { Card, type CardProps } from 'antd'

import styles from '@/components/components.module.css'

/** Kartu dengan kepala biru (ASSET LIST, ASSET STATUS, HOURLY TRAFFIC CHART, ...). */
export default function SectionCard({ title, extra, children, className, ...rest }: CardProps & { title: ReactNode }) {
  return (
    <Card size="small" title={title} extra={extra} {...rest} className={className ? `${styles.sectionCard} ${className}` : styles.sectionCard}>
      {children}
    </Card>
  )
}
