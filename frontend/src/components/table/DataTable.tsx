import { useMemo, useState } from 'react'
import { Alert, Button, Card, Empty, Grid, Input, List, Space, Table, type TableColumnsType } from 'antd'
import { Download, RefreshCw, Search } from 'lucide-react'

import { errorMessage } from '@/services/http/ApiError'
import type { DataTableProps } from '@/types/table.types'
import { downloadExcel } from '@/utils/excel'
import { text } from '@/utils/format'
import styles from '@/components/components.module.css'

import { valueOf } from './columns'

const keyOf = <T,>(rowKey: DataTableProps<T>['rowKey'], row: T) =>
  typeof rowKey === 'function' ? rowKey(row) : String(row[rowKey])

/** Tabel standar: pencarian, urut, halaman, ekspor Excel, galat + coba lagi, tampilan kartu di HP. */
export default function DataTable<T>(p: DataTableProps<T>) {
  const screens = Grid.useBreakpoint()
  const mobile = screens.md === false
  const [q, setQ] = useState('')
  const [exporting, setExporting] = useState(false)

  const filtered = useMemo(() => {
    const rows = p.rows ?? []
    const term = q.trim().toLowerCase()
    if (!term) return rows
    return rows.filter((r) => p.columns.some((c) => String(valueOf(c, r) ?? '').toLowerCase().includes(term)))
  }, [p.rows, p.columns, q])

  const doExport = async () => {
    setExporting(true)
    try {
      const cols = p.columns.filter((c) => c.exportable !== false)
      await downloadExcel(p.exportName!, [{ name: p.exportName!, rows: filtered as never[], columns: cols.map((c) => ({ header: c.title, numFmt: c.numFmt, value: (r: never) => valueOf(c, r as T) })) }])
    } finally {
      setExporting(false)
    }
  }

  const toolbar = (
    <div className={styles.tableToolbar}>
      <Space wrap>
        {p.searchable && (
          <Input allowClear prefix={<Search size={14} />} placeholder="Cari..." value={q} onChange={(e) => setQ(e.target.value)} style={{ width: mobile ? '100%' : 240 }} />
        )}
        {p.toolbar}
      </Space>
      <Space wrap>
        {p.onRetry && <Button icon={<RefreshCw size={14} />} onClick={p.onRetry} loading={p.loading}>Refresh</Button>}
        {p.exportName && (
          <Button icon={<Download size={14} />} onClick={doExport} loading={exporting} disabled={!filtered.length}>Export Excel</Button>
        )}
      </Space>
    </div>
  )

  const errorBox = p.error ? (
    <Alert type="error" showIcon title={errorMessage(p.error)} action={p.onRetry && <Button size="small" onClick={p.onRetry}>Coba lagi</Button>} style={{ marginBottom: 12 }} />
  ) : null

  if (mobile) {
    const titleCol = p.columns.find((c) => c.mobile === 'title') ?? p.columns.find((c) => c.key !== '__no')
    const fields = p.columns.filter((c) => c !== titleCol && c.mobile !== 'hide')
    return (
      <Card size="small">
        {toolbar}
        {errorBox}
        <List
          loading={p.loading}
          dataSource={filtered}
          rowKey={(r) => keyOf(p.rowKey, r)}
          pagination={filtered.length > (p.pageSize ?? 25) ? { pageSize: p.pageSize ?? 25, size: 'small', align: 'center' } : false}
          locale={{ emptyText: <Empty description="Tidak ada data" /> }}
          renderItem={(r, i) => (
            <Card size="small" className={styles.mobileCard} onClick={() => p.onRowClick?.(r)} style={{ marginBottom: 8 }}>
              {titleCol && <div className={styles.mobileTitle}>{titleCol.render ? titleCol.render(r, i) : text(valueOf(titleCol, r))}</div>}
              {fields.map((c) => (
                <div key={c.key} className={styles.mobileRow}>
                  <span className={styles.mobileLabel}>{c.title}</span>
                  <span className={styles.mobileValue}>{c.render ? c.render(r, i) : text(valueOf(c, r))}</span>
                </div>
              ))}
              {p.actions && <div className={styles.mobileActions} onClick={(e) => e.stopPropagation()}>{p.actions(r)}</div>}
            </Card>
          )}
        />
      </Card>
    )
  }

  const columns: TableColumnsType<T> = p.columns.map((c) => ({
    key: c.key,
    title: c.title,
    width: c.width,
    align: c.align,
    fixed: c.fixed,
    ellipsis: false,
    sorter: c.sortable
      ? (a: T, b: T) => {
          const x = valueOf(c, a)
          const y = valueOf(c, b)
          if (x == null) return -1
          if (y == null) return 1
          return x instanceof Date && y instanceof Date ? x.getTime() - y.getTime() : typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y), 'id')
        }
      : undefined,
    render: (_: unknown, r: T, i: number) => (c.render ? c.render(r, i) : text(valueOf(c, r))),
  }))
  if (p.actions) columns.push({ key: '__actions', title: 'Actions', fixed: 'right', width: 110, align: 'center', render: (_: unknown, r: T) => p.actions!(r) })

  return (
    <Card size="small">
      {toolbar}
      {errorBox}
      <Table<T>
        size="small"
        columns={columns}
        dataSource={filtered}
        rowKey={(r) => keyOf(p.rowKey, r)}
        loading={p.loading}
        scroll={{ x: 'max-content', y: p.scrollY }}
        onRow={p.onRowClick ? (r) => ({ onClick: () => p.onRowClick!(r), style: { cursor: 'pointer' } }) : undefined}
        locale={{ emptyText: <Empty description="Tidak ada data" /> }}
        pagination={{
          defaultPageSize: p.pageSize ?? 25,
          showSizeChanger: true,
          pageSizeOptions: [10, 25, 50, 100],
          showTotal: (total, [from, to]) => `${from}–${to} dari ${total} data`,
        }}
      />
    </Card>
  )
}
