import { useState } from 'react'
import { Alert, App, Button, Descriptions, Modal, Space, Table, Upload } from 'antd'
import { Download, FileSpreadsheet } from 'lucide-react'

import { downloadImportTemplate, uploadImport } from '@/services/api/import.api'
import { ApiError, errorMessage } from '@/services/http/ApiError'
import { useAuthStore } from '@/stores/auth.store'
import type { ImportResult, ImportType, RowIssue } from '@/types/import.types'
import { saveBlob } from '@/utils/excel'

interface Props {
  type: ImportType
  title: string
  open: boolean
  onClose: () => void
  onSaved: () => void
}

/**
 * Unggah Excel standar: unduh template → pilih file → PRATINJAU (diperiksa server, belum disimpan)
 * → Simpan. Ada satu baris salah = tidak ada yang disimpan (lebih aman dari sistem lama yang langsung simpan).
 */
export default function ImportModal({ type, title, open, onClose, onSaved }: Props) {
  const { message } = App.useApp()
  const pid = useAuthStore((s) => s.project?.project_id)
  const [file, setFile] = useState<File>()
  const [result, setResult] = useState<ImportResult>()
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState<'preview' | 'save' | 'template'>()

  const reset = () => {
    setFile(undefined)
    setResult(undefined)
    setError(undefined)
  }

  const send = async (f: File, dryRun: boolean) => {
    const body = new FormData()
    body.append('file', f)
    setBusy(dryRun ? 'preview' : 'save')
    setError(undefined)
    try {
      const res = await uploadImport(type, body, pid, dryRun)
      setResult(res.data)
      if (!dryRun) {
        message.success(`Import berhasil: ${res.data.to_create} baru, ${res.data.to_update} diperbarui`)
        onSaved()
        reset()
        onClose()
      }
    } catch (e) {
      // 422 = ada baris salah; daftar kesalahan tetap dikirim server
      if (e instanceof ApiError && e.data) setResult(e.data as ImportResult)
      setError(errorMessage(e))
    } finally {
      setBusy(undefined)
    }
  }

  const template = async () => {
    setBusy('template')
    try {
      saveBlob(await downloadImportTemplate(type), `template_${type}.xlsx`)
    } catch (e) {
      message.error(errorMessage(e))
    } finally {
      setBusy(undefined)
    }
  }

  const canSave = !!file && !!result && result.invalid_rows === 0 && result.valid_rows > 0

  return (
    <Modal
      open={open}
      title={`Upload Excel — ${title}`}
      width={720}
      onCancel={() => {
        reset()
        onClose()
      }}
      footer={
        <Space>
          <Button onClick={() => { reset(); onClose() }}>Batal</Button>
          <Button type="primary" disabled={!canSave} loading={busy === 'save'} onClick={() => file && send(file, false)}>Simpan {result?.valid_rows ?? ''} baris</Button>
        </Space>
      }
    >
      <Space orientation="vertical" style={{ width: '100%' }} size={12}>
        <Button icon={<Download size={14} />} onClick={template} loading={busy === 'template'}>Download Template</Button>
        <Upload.Dragger
          accept=".xlsx,.xls,.csv"
          maxCount={1}
          showUploadList={false}
          beforeUpload={(f) => {
            setFile(f)
            setResult(undefined)
            send(f, true)
            return false
          }}
          disabled={!!busy}
        >
          <p><FileSpreadsheet size={36} /></p>
          <p>{file ? file.name : 'Klik atau seret file Excel ke sini'}</p>
          <p style={{ color: '#64748b', fontSize: 12 }}>.xlsx / .csv, maks. 5 MB — diperiksa dulu sebelum disimpan</p>
        </Upload.Dragger>

        {busy === 'preview' && <Alert type="info" showIcon title="Memeriksa file..." />}
        {error && <Alert type="error" showIcon title={error} />}
        {result && (
          <>
            <Descriptions size="small" bordered column={{ xs: 2, sm: 3 }} items={[
              { key: 't', label: 'Total baris', children: result.total_rows },
              { key: 'v', label: 'Benar', children: result.valid_rows },
              { key: 'i', label: 'Salah', children: <b style={{ color: result.invalid_rows ? '#dc2626' : undefined }}>{result.invalid_rows}</b> },
              { key: 'c', label: 'Data baru', children: result.to_create },
              { key: 'u', label: 'Diperbarui', children: result.to_update },
              { key: 'd', label: 'Dobel di file', children: result.duplicate_rows },
            ]} />
            {[...result.errors, ...result.warnings].length > 0 && (
              <Table<RowIssue & { kind: string }>
                size="small"
                rowKey={(r, i) => `${r.kind}-${r.row}-${r.column}-${i}`}
                pagination={{ pageSize: 8, size: 'small' }}
                dataSource={[...result.errors.map((e) => ({ ...e, kind: 'Salah' })), ...result.warnings.map((w) => ({ ...w, kind: 'Peringatan' }))]}
                columns={[
                  { title: 'Jenis', dataIndex: 'kind', width: 100 },
                  { title: 'Baris', dataIndex: 'row', width: 70 },
                  { title: 'Kolom', dataIndex: 'column', width: 140 },
                  { title: 'Keterangan', dataIndex: 'message' },
                ]}
              />
            )}
          </>
        )}
      </Space>
    </Modal>
  )
}
