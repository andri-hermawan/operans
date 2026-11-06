// System Data → Map Editor: gambar / ubah / hapus area (polygon) langsung di peta, lalu simpan sebagai peta baru
// lewat jalur unggah GeoJSON yang sama (pratinjau dulu; peta lama tetap tersimpan di riwayat, ADR 0003).
import { useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import L from 'leaflet'
import { Alert, App, Button, Card, Empty, Input, Modal, Popconfirm, Space, Table, Tag, Typography } from 'antd'
import { RotateCcw, Save } from 'lucide-react'

import { uploadGeoImport } from '@/services/api/geoImport.api'
import { ApiError, errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { GeoImportResult } from '@/types/geoImport.types'
import type { EditorLayer, ZoneProps } from '@/types/zoneEditor.types'
import { useCan } from '@/hooks/usePermission'
import { useZones } from '@/hooks/useZones'
import { incomplete, styleOf, toFeatureCollection } from '@/utils/zoneEditor'
import BaseMap from '@/components/map/BaseMap'
import GeoImportPreview from '@/components/map/GeoImportPreview'
import MapShell from '@/components/map/MapShell'
import { FitTo } from '@/components/map/layers'
import ZoneEditorLayer from '@/components/map/ZoneEditorLayer'
import PageHeader from '@/components/ui/PageHeader'

import ZonePropsForm from './components/ZonePropsForm'

export default function MapEditorPage() {
  const { message } = App.useApp()
  const qc = useQueryClient()
  const pid = useAuthStore((s) => s.project?.project_id)
  const canEdit = useCan('master', 'import')
  const { data: zones, isFetching } = useZones()

  const [group] = useState(() => L.featureGroup())
  const [loadKey, setLoadKey] = useState(0)
  const [, setVersion] = useState(0) // naik setiap ada perubahan → daftar & form dihitung ulang
  const [dirty, setDirty] = useState(false)
  const [selected, setSelected] = useState<number>()
  const [segFilter, setSegFilter] = useState('')
  const [note, setNote] = useState('')
  const [preview, setPreview] = useState<{ file: File; result: GeoImportResult }>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const layers = group.getLayers() as EditorLayer[]
  const current = layers.find((l) => L.stamp(l) === selected)

  const onChange = useCallback((userEdit: boolean) => {
    setVersion((v) => v + 1)
    setDirty(userEdit)
  }, [])

  const edit = (patch: Partial<ZoneProps>) => {
    if (!current) return
    Object.assign(current.zoneProps, patch)
    current.setStyle(styleOf(current.zoneProps, true))
    onChange(true)
  }

  /** Ambil peta berlaku terbaru dari server, lalu gambar ulang di editor (perubahan yang belum disimpan dibuang). */
  const reload = () => {
    qc.invalidateQueries({ queryKey: [pid, 'zones'] })
    setLoadKey((k) => k + 1)
  }

  const zoomTo = (l: EditorLayer) => {
    setSelected(L.stamp(l))
    const map = (l as unknown as { _map?: L.Map })._map
    map?.fitBounds(l.getBounds(), { maxZoom: 19, padding: [40, 40] })
  }

  const rows = layers.map((l) => ({ id: L.stamp(l), layer: l, ...l.zoneProps }))
  const shown = rows.filter((r) => r.Segment.toLowerCase().includes(segFilter.trim().toLowerCase()))

  /** Simpan = rakit GeoJSON → periksa di server (pratinjau) → baru berlaku setelah dikonfirmasi. */
  const save = async () => {
    const bad = incomplete(layers)
    if (bad.length) {
      zoomTo(bad[0])
      message.error(`${bad.length} area belum punya Category / Segment — lengkapi dulu`)
      return
    }
    if (!layers.length) {
      message.error('Belum ada area. Gambar minimal satu polygon.')
      return
    }
    const file = new File([JSON.stringify(toFeatureCollection(layers))], `map-editor-${new Date().toISOString().slice(0, 10)}.geojson`, { type: 'application/geo+json' })
    await send(file, true)
  }

  const send = async (file: File, dryRun: boolean) => {
    const body = new FormData()
    body.append('file', file)
    if (note) body.append('note', note)
    setBusy(true)
    setError(undefined)
    try {
      const res = await uploadGeoImport(pid, body, dryRun)
      if (dryRun) {
        setPreview({ file, result: res.data })
        return
      }
      message.success(`Peta baru berlaku: ${res.data.valid_features} area`)
      qc.invalidateQueries({ queryKey: [pid, 'geo-imports'] })
      qc.invalidateQueries({ queryKey: [pid, 'zones'] })
      setPreview(undefined)
      setNote('')
      setDirty(false)
    } catch (e) {
      if (e instanceof ApiError && e.data) setPreview({ file, result: e.data as GeoImportResult })
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader
        title="Map Editor"
        subtitle="Gambar, ubah, atau hapus area (polygon) langsung di peta. Hasilnya disimpan sebagai peta baru (peta lama tetap di riwayat)."
        extra={
          canEdit && (
            <Space>
              {/* selalu aktif: tanpa perubahan langsung memuat ulang; ada perubahan → minta konfirmasi dulu */}
              <Popconfirm title="Buang semua perubahan dan muat ulang peta yang berlaku?" disabled={!dirty} okText="Muat ulang" onConfirm={reload}>
                <Button icon={<RotateCcw size={14} />} loading={isFetching} onClick={dirty ? undefined : reload}>Muat ulang</Button>
              </Popconfirm>
              <Button type="primary" icon={<Save size={14} />} loading={busy && !preview} disabled={!dirty} onClick={save}>Simpan peta…</Button>
            </Space>
          )
        }
      />
      {!canEdit && <Alert type="info" showIcon style={{ marginBottom: 12 }} title="Hanya admin project (atau superadmin) yang boleh mengubah peta. Anda hanya bisa melihat." />}
      {dirty && <Alert type="warning" showIcon style={{ marginBottom: 12 }} title="Ada perubahan yang belum disimpan. Peta baru berlaku setelah Anda menekan “Simpan peta…” dan mengonfirmasi." />}
      <MapShell
        panelWidth="440px"
        map={(open) => (
          <BaseMap resizeKey={open}>
            <ZoneEditorLayer group={group} zones={zones} loadKey={loadKey} editable={canEdit} selected={selected} onSelect={setSelected} onChange={onChange} />
            <FitTo area={zones} fitKey={`${loadKey}-${zones?.features.length ?? 0}`} />
          </BaseMap>
        )}
        panel={
          <>
      <Card size="small" title={current ? `Properti area${current.zoneProps.Segment ? ` · ${current.zoneProps.Segment}` : ''}` : 'Properti area'}>
        {current ? (
          canEdit ? (
            <ZonePropsForm layer={current} layers={layers} onChange={edit} />
          ) : (
            <Typography.Text>{current.zoneProps.Category} · {current.zoneProps.Segment}</Typography.Text>
          )
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={canEdit ? 'Klik sebuah area di peta atau daftar, atau gambar polygon baru (ikon di kiri atas peta).' : 'Klik sebuah area untuk melihat propertinya.'} />
        )}
      </Card>
      <Card size="small" title={`Daftar area · ${shown.length === rows.length ? rows.length : `${shown.length} dari ${rows.length}`}`}>
        <Input.Search allowClear placeholder="Filter segment…" value={segFilter} onChange={(e) => setSegFilter(e.target.value)} style={{ marginBottom: 8 }} />
        <Table
            size="small"
            rowKey="id"
            pagination={{ pageSize: 8, size: 'small' }}
            dataSource={shown}
            rowSelection={{ type: 'radio', selectedRowKeys: selected == null ? [] : [selected], columnWidth: 28, onChange: (_k, [r]) => r && zoomTo(r.layer) }}
            onRow={(r) => ({ onClick: () => zoomTo(r.layer) })}
            columns={[
              { title: 'Segment', dataIndex: 'Segment', render: (v: string) => v || <Typography.Text type="danger">(kosong)</Typography.Text> },
              { title: 'Category', dataIndex: 'Category', render: (v: string, r) => (v ? <Tag color={r.HEX_Fill}><span style={{ color: '#1f2937' }}>{v}</span></Tag> : <Typography.Text type="danger">(kosong)</Typography.Text>) },
            ]}
          />
      </Card>
          </>
        }
      />

      <Modal
        open={!!preview}
        title="Simpan peta baru"
        width={680}
        onCancel={() => setPreview(undefined)}
        footer={
          <Space>
            <Button onClick={() => setPreview(undefined)}>Kembali ke editor</Button>
            <Button type="primary" loading={busy} disabled={!preview || preview.result.invalid_features > 0 || !preview.result.valid_features} onClick={() => preview && send(preview.file, false)}>
              Berlakukan peta ini
            </Button>
          </Space>
        }
        destroyOnHidden
      >
        <Space orientation="vertical" style={{ width: '100%' }} size={12}>
          <Input placeholder="Catatan (opsional), mis. tambah area parkir" value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          {error && <Alert type="error" showIcon title={error} />}
          {preview && <GeoImportPreview result={preview.result} />}
        </Space>
      </Modal>
    </>
  )
}
