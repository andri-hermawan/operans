// Hasil pemeriksaan (pratinjau) peta area sebelum diberlakukan — dipakai halaman Map Area dan Map Editor.
import { Descriptions, Table, Tag } from 'antd'

import type { GeoImportResult, GeoIssue } from '@/types/geoImport.types'
import { formatNumber } from '@/utils/format'

export default function GeoImportPreview({ result }: { result: GeoImportResult }) {
  const issues = [...result.errors.map((e) => ({ ...e, kind: 'Salah' })), ...result.warnings.map((w) => ({ ...w, kind: 'Peringatan' }))]
  return (
    <>
      <Descriptions
        size="small"
        bordered
        column={{ xs: 2, sm: 3 }}
        items={[
          { key: 'f', label: 'Area di file', children: result.feature_count },
          { key: 'v', label: 'Benar', children: result.valid_features },
          { key: 'i', label: 'Salah', children: <b style={{ color: result.invalid_features ? '#dc2626' : undefined }}>{result.invalid_features}</b> },
          { key: 'x', label: 'Diperbaiki otomatis', children: result.fixed_geometries },
          { key: 'r', label: 'Area lama diganti', children: formatNumber(result.replaced_zones) },
          { key: 'c', label: 'Kategori', children: result.categories.map((c) => <Tag key={c.category}>{c.category} ({c.count})</Tag>) },
        ]}
      />
      {issues.length > 0 && (
        <Table<GeoIssue & { kind: string }>
          size="small"
          rowKey={(r, i) => `${r.kind}-${i}`}
          pagination={{ pageSize: 6, size: 'small' }}
          dataSource={issues}
          columns={[
            { title: 'Jenis', dataIndex: 'kind', width: 100 },
            { title: 'Area ke-', dataIndex: 'feature', width: 80 },
            { title: 'Nama', dataIndex: 'name' },
            { title: 'Keterangan', dataIndex: 'message' },
          ]}
        />
      )}
    </>
  )
}
