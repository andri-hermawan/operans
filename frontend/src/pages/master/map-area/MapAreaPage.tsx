// System Data → Map Area: unggah peta area (GeoJSON) project — pratinjau dulu, lalu berlaku.
// Peta lama tidak dihapus: tetap dipakai untuk data lampau (geo.zones berversi, ADR 0003).
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Alert, App, Button, Card, Input, Modal, Space, Table, Tag, Upload } from 'antd'
import { Download, FileJson, Upload as UploadIcon } from 'lucide-react'

import { downloadGeoImportFile, uploadGeoImport } from '@/services/api/geoImport.api'
import { ApiError, errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { GeoImportResult, LayerImport } from '@/types/geoImport.types'
import { useGeoImports } from '@/hooks/useGeoImports'
import { useCan } from '@/hooks/usePermission'
import { useZones } from '@/hooks/useZones'
import { saveBlob } from '@/utils/excel'
import { formatDateTime } from '@/utils/format'
import BaseMap from '@/components/map/BaseMap'
import GeoImportPreview from '@/components/map/GeoImportPreview'
import MapShell from '@/components/map/MapShell'
import { FitTo, ZoneLayer } from '@/components/map/layers'
import PageHeader from '@/components/ui/PageHeader'

export default function MapAreaPage() {
  const { message } = App.useApp()
  const qc = useQueryClient()
  const pid = useAuthStore((s) => s.project?.project_id)
  const canUpload = useCan('master', 'import')
  const { data: zones } = useZones()
  const imports = useGeoImports()
  const [open, setOpen] = useState(false)
  const [file, setFile] = useState<File>()
  const [note, setNote] = useState('')
  const [result, setResult] = useState<GeoImportResult>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)

  const send = async (f: File, dryRun: boolean) => {
    const body = new FormData()
    body.append('file', f)
    if (note) body.append('note', note)
    setBusy(true)
    setError(undefined)
    try {
      const res = await uploadGeoImport(pid, body, dryRun)
      setResult(res.data)
      if (!dryRun) {
        message.success(`Peta baru berlaku: ${res.data.valid_features} area`)
        qc.invalidateQueries({ queryKey: [pid, 'geo-imports'] })
        qc.invalidateQueries({ queryKey: [pid, 'zones'] })
        close()
      }
    } catch (e) {
      if (e instanceof ApiError && e.data) setResult(e.data as GeoImportResult)
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const close = () => {
    setOpen(false)
    setFile(undefined)
    setResult(undefined)
    setError(undefined)
    setNote('')
  }
  const download = async (r: LayerImport) => {
    try {
      saveBlob(await downloadGeoImportFile(r.id), r.file_name)
    } catch (e) {
      message.error(errorMessage(e))
    }
  }

  return (
    <>
      <PageHeader
        title="Map Area"
        subtitle="Peta area (segmen jalan, CSA, jembatan timbang, …) yang dipakai untuk zona, status muatan & geofence."
        extra={canUpload && <Button type="primary" icon={<UploadIcon size={14} />} onClick={() => setOpen(true)}>Upload GeoJSON</Button>}
      />
      <MapShell
        panelWidth="440px"
        map={(panelOpen) => (
          <BaseMap resizeKey={panelOpen}>
            <ZoneLayer zones={zones} />
            <FitTo area={zones} fitKey={String(zones?.features.length ?? 0)} />
          </BaseMap>
        )}
        panel={
          <Card size="small" title={`Riwayat Upload · ${zones?.features.length ?? 0} area berlaku`}>
            <Table<LayerImport>
              size="small"
              rowKey="id"
              loading={imports.isFetching}
              dataSource={imports.data}
              pagination={{ pageSize: 8, size: 'small' }}
              scroll={{ x: 'max-content' }}
              columns={[
                { title: 'File', dataIndex: 'file_name', render: (v: string, r) => <Space orientation="vertical" size={0}><b>{v}</b>{r.is_current && <Tag color="success">Berlaku</Tag>}</Space> },
                { title: 'Area', dataIndex: 'feature_count', align: 'right' },
                { title: 'Oleh / Waktu', dataIndex: 'imported_at', render: (v: string, r) => <span>{r.imported_by_name ?? '-'}<br />{formatDateTime(v)}</span> },
                { title: 'Catatan', dataIndex: 'note' },
                { title: '', key: 'dl', render: (_: unknown, r) => <Button size="small" type="text" icon={<Download size={15} />} onClick={() => download(r)} aria-label="Unduh" /> },
              ]}
            />
          </Card>
        }
      />

      <Modal
        open={open}
        title="Upload peta area (GeoJSON)"
        width={680}
        onCancel={close}
        footer={
          <Space>
            <Button onClick={close}>Batal</Button>
            <Button type="primary" loading={busy && !!result} disabled={!file || !result || result.invalid_features > 0 || !result.valid_features} onClick={() => file && send(file, false)}>
              Berlakukan peta ini
            </Button>
          </Space>
        }
      >
        <Space orientation="vertical" style={{ width: '100%' }} size={12}>
          <Input placeholder="Catatan (opsional), mis. revisi jalan Km 13" value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          <Upload.Dragger accept=".geojson,.json" maxCount={1} showUploadList={false} disabled={busy} beforeUpload={(f) => { setFile(f); setResult(undefined); send(f, true); return false }}>
            <p><FileJson size={36} /></p>
            <p>{file ? file.name : 'Klik atau seret file .geojson ke sini'}</p>
            <p style={{ color: '#64748b', fontSize: 12 }}>Diperiksa dulu (geometri, kategori, nama) — belum berlaku sebelum Anda menekan "Berlakukan".</p>
          </Upload.Dragger>
          {error && <Alert type="error" showIcon title={error} />}
          {result && <GeoImportPreview result={result} />}
        </Space>
      </Modal>
    </>
  )
}
