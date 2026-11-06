// Unggah dokumen SOP (PDF) — hanya admin. Teks PDF dibaca server per halaman.
import { useState } from 'react'
import { App, Checkbox, Form, Input, Select, Upload, type UploadFile } from 'antd'
import { Inbox } from 'lucide-react'

import FormDrawer from '@/components/ui/FormDrawer'
import { useDepartments } from '@/hooks/useDepartments'
import { useDocumentActions } from '@/hooks/useDocumentActions'
import { errorMessage } from '@/services/http/ApiError'
import type { ReaderRole } from '@/types/aiDocument.types'
import { READER_ROLE_LABEL } from '@/utils/documents'

const MAX_MB = 25 // sama dengan MAX_UPLOAD_MB di layanan AI Brain

interface Values {
  department_id: string
  title: string
  document_number?: string
  revision?: string
  roles: ReaderRole[]
}

interface Props {
  open: boolean
  onClose: () => void
}

export default function UploadDocumentDrawer({ open, onClose }: Props) {
  const { message } = App.useApp()
  const [form] = Form.useForm<Values>()
  const { data: departments = [], isLoading } = useDepartments()
  const { upload } = useDocumentActions()
  const [file, setFile] = useState<File>()

  const close = () => {
    form.resetFields()
    setFile(undefined)
    onClose()
  }

  const submit = async (v: Values) => {
    if (!file) return message.error('Pilih file PDF terlebih dahulu')
    try {
      const doc = await upload.mutateAsync({ ...v, file })
      if (doc.processing_status === 'processing') message.info(`“${doc.title}” diunggah; teks sedang dibaca (OCR), status diperbarui otomatis`)
      else message.success(`“${doc.title}” diunggah (${doc.page_count} halaman)`)
      close()
    } catch (e) {
      message.error(errorMessage(e))
    }
  }

  const files: UploadFile[] = file ? [{ uid: '1', name: file.name, status: 'done' }] : []

  return (
    <FormDrawer open={open} title="Tambah Dokumen" onClose={close} onSubmit={() => form.submit()} saving={upload.isPending}>
      <Form<Values>
        form={form}
        layout="vertical"
        onFinish={submit}
        initialValues={{ roles: ['dispatcher', 'viewer'] }}
      >
        <Form.Item label="File PDF" required extra={`Maksimal ${MAX_MB} MB. PDF hasil scan dibaca dengan OCR bila diaktifkan di server (prosesnya bisa beberapa menit).`}>
          <Upload.Dragger
            accept=".pdf,application/pdf"
            maxCount={1}
            fileList={files}
            beforeUpload={(f) => {
              if (f.size > MAX_MB * 1024 * 1024) message.error(`Ukuran file maksimal ${MAX_MB} MB`)
              else setFile(f)
              return false // jangan unggah otomatis; dikirim bersama isian saat Simpan
            }}
            onRemove={() => setFile(undefined)}
          >
            <p><Inbox size={32} /></p>
            <p>Klik atau seret file PDF ke sini</p>
          </Upload.Dragger>
        </Form.Item>
        <Form.Item name="title" label="Judul" rules={[{ required: true, message: 'Judul wajib diisi' }, { max: 200 }]}>
          <Input placeholder="mis. SOP Pengoperasian Dump Truck" />
        </Form.Item>
        <Form.Item name="department_id" label="Department" rules={[{ required: true, message: 'Pilih department' }]}
          extra={!isLoading && !departments.length ? 'Belum ada department. Tambahkan lewat tombol “Kelola Department”.' : undefined}>
          <Select loading={isLoading} placeholder="Pilih department" options={departments.map((d) => ({ value: d.id, label: d.department_name }))} />
        </Form.Item>
        <Form.Item name="document_number" label="Nomor dokumen" rules={[{ max: 100 }]}>
          <Input placeholder="opsional" />
        </Form.Item>
        <Form.Item name="revision" label="Revisi" rules={[{ max: 30 }]}>
          <Input placeholder="opsional, mis. Rev. 02" />
        </Form.Item>
        <Form.Item name="roles" label="Boleh dibaca oleh" extra="Admin project selalu boleh membaca.">
          <Checkbox.Group options={(Object.keys(READER_ROLE_LABEL) as ReaderRole[]).map((r) => ({ value: r, label: READER_ROLE_LABEL[r] }))} />
        </Form.Item>
      </Form>
    </FormDrawer>
  )
}
