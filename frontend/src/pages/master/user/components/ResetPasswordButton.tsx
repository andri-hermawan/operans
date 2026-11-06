import { useState } from 'react'
import { App, Button, Input, Modal, Tooltip, Typography } from 'antd'
import { KeyRound } from 'lucide-react'

import { resetUserPassword } from '@/services/api/user.api'
import { errorMessage } from '@/services/http/ApiError'
import type { UserRow } from '@/types/user.types'

export default function ResetPasswordButton({ user }: { user: UserRow }) {
  const { message } = App.useApp()
  const [open, setOpen] = useState(false)
  const [pw, setPw] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    setBusy(true)
    try {
      await resetUserPassword(user.id, pw)
      message.success(`Password ${user.name} direset`)
      setOpen(false)
      setPw('')
    } catch (e) {
      message.error(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <Tooltip title="Reset password"><Button size="small" type="text" icon={<KeyRound size={15} />} onClick={() => setOpen(true)} /></Tooltip>
      <Modal open={open} title={`Reset password — ${user.name}`} okText="Reset" confirmLoading={busy} okButtonProps={{ disabled: pw.length < 8 }} onOk={submit} onCancel={() => setOpen(false)} destroyOnHidden>
        <Input.Password placeholder="Password baru (min. 8 karakter)" value={pw} onChange={(e) => setPw(e.target.value)} autoComplete="new-password" />
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>Semua sesi user ini akan berakhir.</Typography.Text>
      </Modal>
    </>
  )
}
