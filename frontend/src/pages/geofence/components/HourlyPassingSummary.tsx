// Hourly Passing Summary (gaya sistem lama): tabel Hour | In | Out (hanya jam yang ada kejadian); footer Hours + Total.
import { Table, Tag } from 'antd'

import styles from '@/components/monitoring/monitoring.module.css'
import SectionCard from '@/components/ui/SectionCard'

export interface HourlyRow {
  hour: string
  in: number
  out: number
}

export default function HourlyPassingSummary({ rows, emptyText = 'Belum ada data per jam' }: { rows: HourlyRow[]; emptyText?: string }) {
  const tag = (color: string) => (v: number) => <Tag color={color} style={{ margin: 0, width: '100%', textAlign: 'center' }}>{v}</Tag>
  return (
    <SectionCard title="Hourly Passing Summary" className={`${styles.fillCard} ${styles.fillCol}`} styles={{ body: { padding: 0 } }}>
      <div className={styles.scrollBox}>
        <Table
          size="small"
          rowKey="hour"
          dataSource={rows}
          pagination={false}
          tableLayout="fixed"
          sticky
          locale={{ emptyText }}
          columns={[
            { title: 'Hour', dataIndex: 'hour', width: '34%', align: 'center' },
            { title: 'In', dataIndex: 'in', align: 'center', render: tag('green') },
            { title: 'Out', dataIndex: 'out', align: 'center', render: tag('red') },
          ]}
        />
      </div>
      <div className={styles.cardFooter}>
        <span>Hours : <b>{rows.length}</b></span>
        <span>Total : <b>{rows.reduce((s, r) => s + r.in + r.out, 0)}</b></span>
      </div>
    </SectionCard>
  )
}
