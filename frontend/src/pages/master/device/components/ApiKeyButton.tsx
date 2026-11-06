import { useState } from 'react'
import { Alert, App, Button, Modal, Tooltip, Typography } from 'antd'
import { KeyRound } from 'lucide-react'

import { rotateDeviceApiKey } from '@/services/api/device.api'
import { errorMessage } from '@/services/http/ApiError'
import type { Device } from '@/types/device.types'

/** Buat API key baru untuk alat HTTP/simulator — ditampilkan SEKALI. */
export default function ApiKeyButton({ device }: { device: Device }) {
  const { modal, message } = App.useApp()
  const [key, setKey] = useState<string>()
  const rotate = () =>
    modal.confirm({
      title: `Buat API key baru untuk ${device.imei}?`,
      content: device.has_api_key ? 'Key lama langsung tidak berlaku.' : 'Dipakai alat yang mengirim lewat HTTP (simulator). Teltonika TCP tidak memerlukan key.',
      okText: 'Buat key',
      onOk: async () => {
        try {
          setKey(await rotateDeviceApiKey(device.id))
        } catch (e) {
          message.error(errorMessage(e))
          throw e
        }
      },
    })
  return (
    <>
      <Tooltip title="API key (HTTP)"><Button size="small" type="text" icon={<KeyRound size={15} />} onClick={rotate} /></Tooltip>
      <Modal open={!!key} title="API key baru" footer={<Button type="primary" onClick={() => setKey(undefined)}>Sudah disimpan</Button>} closable={false}>
        <Alert type="warning" showIcon title="Simpan sekarang — key ini tidak bisa ditampilkan lagi." style={{ marginBottom: 12 }} />
        <Typography.Paragraph code copyable={{ text: key }} style={{ wordBreak: 'break-all' }}>{key}</Typography.Paragraph>
      </Modal>
    </>
  )
}
