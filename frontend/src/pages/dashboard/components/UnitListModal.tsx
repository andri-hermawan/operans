// Daftar unit di balik sebuah angka Dashboard: Asset ID (equipment_code), tipe, status, dan data terakhir.
import { Modal, Table, Tag, Typography } from 'antd'

import type { UnitListItem } from '@/types/dashboard.types'
import { equipmentStatusColor, equipmentStatusLabel } from '@/utils/colors'
import { formatDateTime } from '@/utils/format'

interface Props {
  open: boolean
  onClose: () => void
  title: string
  /** Penjelasan singkat dari mana daftar ini dihitung. */
  description: string
  units: UnitListItem[]
  emptyText?: string
}

export default function UnitListModal({ open, onClose, title, description, units, emptyText = 'Tidak ada asset' }: Props) {
  return (
    <Modal open={open} onCancel={onClose} footer={null} width={640} title={`${title} (${units.length})`} destroyOnHidden>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>{description}</Typography.Paragraph>
      <Table<UnitListItem>
        size="small"
        rowKey="id"
        dataSource={units}
        pagination={units.length > 12 ? { pageSize: 12, showSizeChanger: false } : false}
        locale={{ emptyText }}
        columns={[
          { title: 'No', width: 52, render: (_: unknown, __: UnitListItem, i: number) => i + 1 },
          { title: 'Asset ID', dataIndex: 'code', sorter: (a, b) => a.code.localeCompare(b.code), render: (c: string) => <b>{c}</b> },
          { title: 'Tipe', dataIndex: 'type', render: (t: string | null) => t ?? '-' },
          {
            title: 'Status',
            dataIndex: 'status',
            render: (s: string | null) =>
              s ? <Tag color={equipmentStatusColor[s]} variant="solid">{equipmentStatusLabel[s] ?? s}</Tag> : <Tag>Belum pernah terpantau</Tag>,
          },
          { title: 'Data Terakhir', dataIndex: 'lastEventAt', render: (t: string | null) => (t ? formatDateTime(t) : '-') },
        ]}
      />
    </Modal>
  )
}
