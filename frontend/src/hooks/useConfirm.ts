import { App } from 'antd'

import { errorMessage } from '@/services/http/ApiError'

/**
 * Konfirmasi aksi berbahaya (hapus, kirim, ...) dengan pesan berhasil/gagal yang seragam.
 *   const confirm = useConfirm()
 *   confirm({ title: 'Hapus unit DT01?', action: () => api.delete(id), success: 'Unit dihapus' })
 */
export function useConfirm() {
  const { modal, message } = App.useApp()
  return ({ title, content, okText = 'Ya, Hapus', danger = true, action, success }: {
    title: string
    content?: string
    okText?: string
    danger?: boolean
    action: () => Promise<unknown>
    success?: string
  }) =>
    modal.confirm({
      title,
      content,
      okText,
      cancelText: 'Batal',
      okButtonProps: { danger },
      onOk: async () => {
        try {
          await action()
          if (success) message.success(success)
        } catch (e) {
          message.error(errorMessage(e))
          throw e // dialog tetap terbuka
        }
      },
    })
}
