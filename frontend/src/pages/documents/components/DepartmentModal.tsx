// Kelola department (pengelompokan dokumen per project) — hanya admin: lihat daftar & tambah.
import { App, Button, Form, Input, List, Modal } from 'antd'

import { useDepartments } from '@/hooks/useDepartments'
import { useDocumentActions } from '@/hooks/useDocumentActions'
import { errorMessage } from '@/services/http/ApiError'
import type { NewDepartment } from '@/types/aiDocument.types'

interface Props {
  open: boolean
  onClose: () => void
}

export default function DepartmentModal({ open, onClose }: Props) {
  const { message } = App.useApp()
  const [form] = Form.useForm<NewDepartment>()
  const { data: departments = [], isLoading } = useDepartments()
  const { createDepartment } = useDocumentActions()

  const add = async (v: NewDepartment) => {
    try {
      const d = await createDepartment.mutateAsync(v)
      message.success(`Department ${d.department_name} ditambahkan`)
      form.resetFields()
    } catch (e) {
      message.error(errorMessage(e))
    }
  }

  return (
    <Modal open={open} title="Kelola Department" onCancel={onClose} footer={<Button onClick={onClose}>Tutup</Button>} destroyOnHidden>
      <List
        size="small"
        bordered
        loading={isLoading}
        dataSource={departments}
        locale={{ emptyText: 'Belum ada department' }}
        renderItem={(d) => <List.Item><b>{d.department_code}</b>&nbsp;— {d.department_name}</List.Item>}
        style={{ marginBottom: 16, maxHeight: 240, overflow: 'auto' }}
      />
      <Form<NewDepartment> form={form} layout="inline" onFinish={add} style={{ rowGap: 8 }}>
        <Form.Item name="department_code" rules={[{ required: true, message: 'Kode?' }, { pattern: /^[A-Za-z0-9_-]+$/, message: 'Huruf/angka/-/_ saja' }]}>
          <Input placeholder="Kode (OPS)" maxLength={30} style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="department_name" rules={[{ required: true, message: 'Nama?' }]}>
          <Input placeholder="Nama department" maxLength={100} style={{ width: 200 }} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={createDepartment.isPending}>Tambah</Button>
      </Form>
    </Modal>
  )
}
