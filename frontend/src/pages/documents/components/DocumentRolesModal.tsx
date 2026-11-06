// Ubah role yang boleh membaca sebuah dokumen (admin).
import { useState } from 'react'
import { App, Checkbox, Modal, Typography } from 'antd'

import { useDocumentActions } from '@/hooks/useDocumentActions'
import { errorMessage } from '@/services/http/ApiError'
import type { AiDocument, ReaderRole } from '@/types/aiDocument.types'
import { READER_ROLE_LABEL } from '@/utils/documents'

interface Props {
  doc: AiDocument | undefined
  onClose: () => void
}

export default function DocumentRolesModal({ doc, onClose }: Props) {
  // key: tiap dokumen memulai pilihan dari hak bacanya sekarang
  return doc ? <RolesDialog key={doc.id} doc={doc} onClose={onClose} /> : null
}

function RolesDialog({ doc, onClose }: { doc: AiDocument; onClose: () => void }) {
  const { message } = App.useApp()
  const { setRoles } = useDocumentActions()
  const [roles, setLocal] = useState<ReaderRole[]>(doc.roles ?? [])

  const save = async () => {
    try {
      await setRoles.mutateAsync({ id: doc.id, roles })
      message.success('Hak baca diperbarui')
      onClose()
    } catch (e) {
      message.error(errorMessage(e))
    }
  }

  return (
    <Modal open title="Hak baca dokumen" onCancel={onClose} onOk={save} okText="Simpan" cancelText="Batal" confirmLoading={setRoles.isPending}>
      <Typography.Paragraph strong>{doc.title}</Typography.Paragraph>
      <Typography.Paragraph type="secondary">Admin project selalu boleh membaca. Tanpa pilihan di bawah, hanya admin yang melihat dokumen ini.</Typography.Paragraph>
      <Checkbox.Group value={roles} onChange={(v) => setLocal(v as ReaderRole[])} options={(Object.keys(READER_ROLE_LABEL) as ReaderRole[]).map((r) => ({ value: r, label: READER_ROLE_LABEL[r] }))} />
    </Modal>
  )
}
