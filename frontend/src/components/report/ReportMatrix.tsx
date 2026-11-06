// Laporan matriks Asset ID × tanggal (Ritase, Payload, Produksi, Breakdown, PA). Satu komponen untuk 5 laporan
// yang di FMS lama ditulis terpisah ±95% sama. Halaman cukup memberi baris, satuan, format, warna & rumus total.
import { useState } from 'react'
import { Button, Card, Empty, Table, type TableColumnsType } from 'antd'
import { Download } from 'lucide-react'

import type { MatrixProps, MatrixRow } from '@/types/report.types'
import { downloadExcel } from '@/utils/excel'
import { dayjs } from '@/utils/time'

export default function ReportMatrix(p: MatrixProps) {
  const [exporting, setExporting] = useState(false)
  const cell = (v: number | null) => ({ children: p.format(v), style: p.cellStyle?.(v) })

  const columns: TableColumnsType<MatrixRow> = [
    { title: 'No', key: 'no', width: 50, fixed: 'left', align: 'center', render: (_v, _r, i) => i + 1 },
    { title: 'Asset ID', dataIndex: 'code', width: 120, fixed: 'left', render: (v: string) => <b>{v}</b> },
    { title: 'UoM', key: 'uom', width: 90, fixed: 'left', render: () => p.uom },
    ...p.days.map((d) => ({
      title: dayjs(d).format('DD-MMM'),
      key: d,
      width: 72,
      align: 'center' as const,
      onCell: (r: MatrixRow) => ({ style: { ...p.cellStyle?.(r.values[d] ?? null), padding: '4px 6px' } }),
      render: (_: unknown, r: MatrixRow) => cell(r.values[d] ?? null).children,
    })),
    {
      title: p.totalTitle,
      key: 'total',
      width: 100,
      fixed: 'right',
      align: 'center',
      onCell: (r: MatrixRow) => ({ style: { ...p.cellStyle?.(r.total), fontWeight: 700 } }),
      render: (_: unknown, r: MatrixRow) => p.format(r.total),
    },
  ]

  const doExport = async () => {
    setExporting(true)
    try {
      const fmt = p.digits ? `0.${'0'.repeat(p.digits)}` : '0'
      await downloadExcel(p.exportName, [{
        name: p.title,
        rows: p.rows as never[],
        columns: [
          { header: 'Asset ID', value: (r: MatrixRow) => r.code },
          { header: 'UoM', value: () => p.uom },
          ...p.days.map((d) => ({ header: dayjs(d).format('DD-MMM'), numFmt: fmt, value: (r: MatrixRow) => r.values[d] ?? null })),
          { header: p.totalTitle, numFmt: fmt, value: (r: MatrixRow) => r.total },
        ],
      }] as never)
    } finally {
      setExporting(false)
    }
  }

  return (
    <Card
      size="small"
      title={p.title}
      extra={<Button size="small" icon={<Download size={14} />} onClick={doExport} loading={exporting} disabled={!p.rows.length}>Export Excel</Button>}
    >
      <Table<MatrixRow>
        size="small"
        bordered
        sticky
        rowKey="code"
        loading={p.loading}
        columns={columns}
        dataSource={p.rows}
        pagination={false}
        scroll={{ x: 'max-content', y: 'calc(100vh - 340px)' }}
        locale={{ emptyText: <Empty description="Tidak ada data pada periode ini" /> }}
      />
    </Card>
  )
}
